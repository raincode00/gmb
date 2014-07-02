$(document).ready(main);

function gen_id(len) {
    len = len || 32;
    var id = "";
    var c = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
    for(var i = 0; i < len; i++) {
        id += c.charAt(Math.floor(Math.random() * c.length));
    }
    return id;
};

function Client(topic) {
    
    var user_id = gen_id()
    //localStorage.user_id = user_id;
    
    var call_api = function(params) {
        return $.ajax({
            url: params.url,
            dataType: 'json',
            contentType: 'application/json; charset=utf-8',
            type: params.type,
            data: JSON.stringify(params.data),
            cache: params.cache || false,
            success: params.success,
            error: params.error
        });
    };
    var attributes = {
        display_name: "Anonymous"
    };
    attributes.display_name = localStorage.display_name || attributes.display_name;
    localStorage.display_name = attributes.display_name;
    
    var active = true;
    var topic_data = {};
    var callback = function(message){
        console.log(message);
    };
    var callback_loop = function() {
        $.when(
            call_api({
                type:"GET",
                url: '/api/topics/' + topic + '/subscribers/' + user_id + '/messages',
            })
        ).done(function(data) {
            $.each(data.messages, function(i, message) {
                console.log(message);
                if ("attributes_updated" in message) {
                    $.when(call_api({
                        type: "GET",
                        url: '/api/subscribers/' + message.attributes_updated
                    })).done(function(data) {
                        topic_data.subscribers[message.attributes_updated] = data;
                    });
                }
                callback(message);
            });
        }).always(function() {
            if (active) callback_loop()
        })
    };
    callback_loop();
    $.when(call_api({
        type: "GET",
        url: '/api/topics/' + topic
    })).done(function(data) {
        topic_data = data;
    });
    var client_object = {
        publish: function(message) {
            return call_api({
                type:"POST",
                url: '/api/topics/' + topic + '/subscribers/' + user_id + '/messages',
                data: {message:message}
            });
        },
        on_message: function(func) {
            callback = func
        },
        user_id: function() {
            return user_id;
        },
        topic_id: function() {
            return topic;
        },
        topic_data: function() {
            return topic_data;
        },
        get_topic: function() {
            return call_api({
                type: "GET",
                url: '/api/topics/' + topic
            }).done(function(data) {
                topic_data = data;
            });
        },
        get_user: function() {
            return call_api({
                type: "GET",
                url: '/api/subscribers/' + user_id
            });
        },
        set_attribute: function(attr, value) {
            attributes[attr] = value;
            localStorage[attr] = value;
            return call_api({
                type:"POST",
                url: '/api/subscribers/' + user_id,
                data: {attributes:attributes}
            }).then(function() {
                client_object.publish({"attributes_updated": user_id})
            });
        }
    };
    client_object.set_attribute("display_name",attributes.display_name);
    return client_object;
}

var client = null;

function render_template(params) {
    var template_name = params.template;
    var query_selector = params.element;
    var data = params.data;
    var append = params.append;
    var html = $("#template-"+template_name).html();
    html = _.template(html, data);
    if (append) {
        $(query_selector).append(html);
    } else {
        $(query_selector).html(html);
    }
}
function render_chat_message(message) {
    var topic_data = client.topic_data();
    var template_data = {
        message: message.message.chat,
        from: topic_data.subscribers[message.from_subscriber].attributes.display_name
    }
    render_template({
        template: "chat-message",
        element: "#chat-messages",
        data: template_data,
        append: true
    })
    $("#chat-messages").scrollTop($("#chat-messages").prop("scrollHeight") - $("#chat-messages").height());
  
}

function parse_hash() {
    if (!window.location.hash) return {};
    hash = window.location.hash.slice(1);
    key_vals = hash.split("!");
    key_vals_parsed = {};
    $.each(key_vals, function(i, key_val) {
        key_val = key_val.split(':');
        key = key_val[0];
        val = key_val[1];
        key_vals_parsed[key] = val || "";
    })
    return key_vals_parsed;
}
function set_hash(key_vals) {
    var hash = "#";
    var has_one = false;
    $.each(key_vals, function(key, val) {
        hash += (has_one?"!":"") + key + ":" + val;
        has_one = true;
    })
    window.location.hash = hash;
}

function send_chat_message() {
    var message = $.trim($("#chat-send-textarea").val());
    if (!message.length) return    
    client.publish({chat:message});
    console.log(message);
    $("#chat-send-textarea").val("");
}

function main() {
    config = parse_hash();
    config.r = config.r || gen_id(16);
    client = Client(config.r);
    set_hash(config);
    client.on_message(function(message) {
        if ("chat" in message.message) {
            render_chat_message(message)
        }
        console.log(message);
    });
    $("#chat-send-button").click(function() {
        send_chat_message();
    });
    $("#chat-send-textarea").keypress(function(e) {
        if (e.which == 13) {
            $("#chat-send-button").click();
            return false;
        }
    });
}