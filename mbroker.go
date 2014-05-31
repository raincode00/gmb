package main

import (
    "fmt"
    "flag"
    "time"
    "net"
    "net/rpc"
    "net/rpc/jsonrpc"
    "github.com/hoisie/web"
    "encoding/json"
    "github.com/nu7hatch/gouuid"
    //"os"
)

type Message struct {
    FromSubscriber string   `json:"from_subscriber"`
    FromTopic string        `json:"from_topic"`
    Content interface{}     `json:"message"`
}

type Subscriber struct {
    SubscriberID string       `json:"subscriber_id"`
    Topics map[string]string  `json:"topics"`
    Attributes map[string]interface{} `json:"attributes"`
    _messages chan *Message
    _subscriberKey string
}

func (u *Subscriber) PostMessage(msg *Message) {
    go func() {
        select {
            case u._messages <- msg:
                return
            case <- time.After(60*time.Second):
                fmt.Println("Time-out while sending:", msg)
                /*for _topic, _ := range u.Topics {
                    topic := server.GetTopic(_topic)
                    //topic.DeleteSubscriber(u.SubscriberID)
                }*/
                return
        }
    }()
}

type Topic struct {
    TopicID string          `json:"topic_id"`
    DisplayName string      `json:"display_name"`
    Subscribers map[string]*Subscriber `json:"subscribers"`
    Messages []*Message     `json:"messages"`
    _messages chan *Message
}

func (topic *Topic) PostMessage(msg *Message) {
    msg.FromTopic = topic.TopicID
    fmt.Println("Post Message:", msg)
    for _, u := range topic.Subscribers {
        u.PostMessage(msg)
    }
    topic.Messages = append(topic.Messages, msg)
    if len(topic.Messages) > 64 {
        topic.Messages = topic.Messages[len(topic.Messages) - 64:]
    }
    /*go func() {
        select {
            case topic._messages <- msg:
                return
            case <- time.After(60*time.Second):
                fmt.Println("Time-out while sending:", msg)
                return
        }
    }()*/
}

func (topic *Topic) AddSubscriber(subscriber *Subscriber) {
    fmt.Println("AddSubscriber", subscriber.SubscriberID)
    subscriber.Topics[topic.TopicID] = topic.TopicID
    topic.Subscribers[subscriber.SubscriberID] = subscriber
    
}

func (topic *Topic) DeleteSubscriber(subscriber_id string) {
    delete(topic.Subscribers, subscriber_id)
}

func (topic *Topic) GetSubscriber(subscriber_id string) *Subscriber {
    subscriber, ok :=  topic.Subscribers[subscriber_id];
    if ok {
        return subscriber
    } else {
        subscriber =  subscribers.GetSubscriber(subscriber_id);
        topic.AddSubscriber(subscriber)
        return subscriber
    }
}

type ChatServer struct {
    Topics map[string]*Topic
}

func (server *ChatServer) CreateTopic(id string) {
    fmt.Println("CreateTopic", id)
    server.Topics[id] = &Topic {
        TopicID: id,
        Subscribers: map[string]*Subscriber {},
        Messages: []*Message {},
        _messages: make(chan *Message),
    }
}

func (server *ChatServer) GetTopic(id string) *Topic {
    topic, ok := server.Topics[id]
    if ok {
        return topic
    } else {
        server.CreateTopic(id)
        return server.Topics[id]
    }
}


func (server *ChatServer) GetTopics() map[string]map[string]string {
    l := map[string]map[string]string{}
    for k, v := range server.Topics {
        l[k] = map[string]string {
            "display_name": v.DisplayName,
            "topic_id": v.TopicID,
        }
    }
    return l
}

func (server *ChatServer) PostTopicMessage(_topic string, subscriber_id string, msg_content interface{}) {
    topic := server.GetTopic(_topic)
    subscriber := topic.GetSubscriber(subscriber_id)
    topic.PostMessage(&Message{
        Content: msg_content,
        FromSubscriber: subscriber.SubscriberID,
    })
}



func (server *ChatServer) GetTopicMessages(_topic string, subscriber_id string) []Message {
    topic := server.GetTopic(_topic)
    subscriber := topic.GetSubscriber(subscriber_id)
    messages := make([]Message,0)
    found := false
    for {
        select {
            case msg := <- subscriber._messages:
                messages = append(messages,*msg)
            case <- time.After(1*time.Millisecond):
                if len(messages) > 0 {
                    found = true
                }
                break
        }
        if found {
            break
        }
    }
 
    return messages
}


type SubscriberDatabase map[string]*Subscriber

func (subscriber_db SubscriberDatabase) GetSubscriber(subscriber_id string) *Subscriber {
    subscriber, ok := subscriber_db[subscriber_id]
    if ok {
        return subscriber
    } else {
        subscriber = &Subscriber{
            SubscriberID: subscriber_id,
            Topics: map[string]string{},
            Attributes: map[string]interface{}{},
            _messages: make(chan *Message),
        }
        subscriber_db[subscriber_id] = subscriber
        return subscriber
    }
}
func APIRequestBody(ctx *web.Context) interface{} {
    var value interface{}
    decoder := json.NewDecoder(ctx.Request.Body)
    decoder.Decode(&value)
    return value
}

func APIResponse(ctx *web.Context, resp  interface{}) string {
    resp_text, _ := json.Marshal(resp)
    ctx.SetHeader("Content-Type", "application/json", true)
    return string(resp_text[:])
}
func APIErrorResponse(ctx *web.Context, message string, status int) {
    resp_text := APIResponse(ctx, map[string]interface{}{"error":message})
    ctx.Abort(status, resp_text)
}


func APIPostTopicMessage(ctx *web.Context, _topic string, subscriber_id string) string {
    topic := server.GetTopic(_topic)
    subscriber := topic.GetSubscriber(subscriber_id)
    if !CheckAuthKeyCookie(ctx, subscriber) {
        APIErrorResponse(ctx, "Invalid SubscriberKey", 401)
        return ""
    }
    req := APIRequestBody(ctx).(map[string]interface{})
    msg_content := req["message"]
    server.PostTopicMessage(_topic, subscriber_id, msg_content)
    return APIResponse(ctx, map[string]interface{}{"success":0})
}


func APIGetTopicMessages(ctx *web.Context, _topic string, subscriber_id string) string {
    return APIResponse(ctx, map[string]interface{}{
        "messages": server.GetTopicMessages(_topic, subscriber_id),
    })
}




func APIGetTopics(ctx *web.Context) string {
    return APIResponse(ctx, map[string]interface{}{
        "topics": server.GetTopics(),
    })
}

func APIGetTopic(ctx *web.Context, _topic string) string {
    topic := server.GetTopic(_topic)
    return APIResponse(ctx, topic)
}

func APIUpdateTopic(ctx *web.Context, _topic string) string {
    topic := server.GetTopic(_topic)
    req := APIRequestBody(ctx).(map[string]interface{})
    if name, ok := req["display_name"].(string); ok {
        topic.DisplayName = name
    }
    return APIResponse(ctx, topic)
}

func APIGetSubscriber(ctx *web.Context, subscriber_id string) string {
    subscriber := subscribers.GetSubscriber(subscriber_id)
    return APIResponse(ctx, subscriber) 
}


func CheckAuthKeyCookie(ctx *web.Context, subscriber *Subscriber) bool {
    return true;
     if len(subscriber._subscriberKey) == 0 {
        key, _ := uuid.NewV4()
        ctx.SetCookie(web.NewCookie("SubscriberKey", key.String(),0))
        fmt.Println(key)
        subscriber._subscriberKey = key.String()
        return true
    }
    if key, err := ctx.Request.Cookie("SubscriberKey"); err == nil {
        
        return key.Value == subscriber._subscriberKey
    } else {
        fmt.Println(err)
    }
    return false
}

func APIUpdateSubscriber(ctx *web.Context, subscriber_id string) string {
    subscriber := subscribers.GetSubscriber(subscriber_id)
    req := APIRequestBody(ctx).(map[string]interface{})
    if !CheckAuthKeyCookie(ctx, subscriber) {
        APIErrorResponse(ctx, "Invalid SubscriberKey", 401)
        return ""
    }
    if attr, ok := req["attributes"].(map[string]interface{}); ok {
        subscriber.Attributes = attr
    }
    return APIResponse(ctx, subscriber) 
}


type RPCService struct{}

func (rpcservice *RPCService) RPCGetTopics(request interface{}, response *map[string]map[string]string) error {
    *response = server.GetTopics()
    return nil
}

func (rpcservice *RPCService) RPCGetTopicMessages(request *map[string]string, response *[]Message) error {
    *response = server.GetTopicMessages((*request)["topic_id"], (*request)["subscriber_id"])
    return nil
}
func (rpcservice *RPCService) RPCPostTopicMessage(request *map[string]interface{}, response *interface{}) error {
    fmt.Println(*request)
    server.PostTopicMessage((*request)["topic_id"].(string), (*request)["subscriber_id"].(string), (*request)["message"])
    *response = "success"
    return nil
}


type RPCClient struct {
    client *rpc.Client
}

func (client  *RPCClient) Connect(address string) {
    client.client , _ = jsonrpc.Dial("tcp", address)
}

func (client  *RPCClient) PostTopicMessage(topic_id string, subscriber_id string, message interface{}) {
    request := map[string]interface{} {
        "topic_id":topic_id,
        "subscriber_id": subscriber_id,
        "message": message,
    }
    client.client.Call(
        "RPCService.RPCPostTopicMessage", 
        &request, 
        new(interface{}),
    )
}

func (client  *RPCClient) GetTopicMessages(topic_id string, subscriber_id string) []Message {
    request := map[string]string {
        "topic_id": topic_id,
        "subscriber_id": subscriber_id,
    }
    var response []Message
    client.client.Call(
        "RPCService.RPCGetTopicMessages", 
        &request, 
        &response,
    )
    return response
}


var server = ChatServer{
    Topics: map[string]*Topic{},
}
var subscribers = SubscriberDatabase{}

func main() {  
    bind_addr := flag.String("bind_ip", "127.0.0.1", "bind ip address")
    http_port := flag.Int("http_port", 9999, "listen http port")
    rpc_port := flag.Int("rpc_port", 9998, "listen rpc port")
    flag.Parse()
    
    go func() {
        addr, _ := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d",*bind_addr,*rpc_port))
        listener, _ := net.ListenTCP("tcp", addr)
        rpcservice := new(RPCService)
        rpc.Register(rpcservice)
        rpc.HandleHTTP()
        for {
            conn, _ := listener.Accept()
            go rpc.ServeCodec(jsonrpc.NewServerCodec(conn))
        }
    }()
    
    web.Get("/api/topics/([a-zA-Z0-9_\\-]+)/subscribers/([a-zA-Z0-9_\\-]+)/messages", APIGetTopicMessages)
    web.Post("/api/topics/([a-zA-Z0-9_\\-]+)/subscribers/([a-zA-Z0-9_\\-]+)/messages", APIPostTopicMessage)
    web.Get("/api/topics/([a-zA-Z0-9_\\-]+)", APIGetTopic)
    web.Post("/api/topics/([a-zA-Z0-9_\\-]+)", APIUpdateTopic)
    //web.Get("/api/topics", APIGetTopics)
    web.Get("/api/subscribers/([a-zA-Z0-9_\\-]+)", APIGetSubscriber)
    web.Post("/api/subscribers/([a-zA-Z0-9_\\-]+)", APIUpdateSubscriber)
    //web.Get("/api/topics/(.+)/subscribers/(.+)", APIGetTopicSubscriber)
    //web.Get("/api/topics/(.+)/subscribers", APIGetTopicSubscribers)
    
    
    web.Run(fmt.Sprintf("%s:%d",*bind_addr,*http_port))
    
}   