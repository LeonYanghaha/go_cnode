package reply

import (
	"log"
	"net/http"
	//"regexp"

	"go_cnode/mgoModels"
	//"github.com/dangyanglim/go_cnode/service/mail"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/tommy351/gin-sessions"
	"go_cnode/service/cache"
	"regexp"
)

var userModel = new(models.UserModel)
var topicModel = new(models.TopicModel)
var replyModel = new(models.ReplyModel)
var messageModel = new(models.MessageModel)

func ShowCreate(c *gin.Context) {
	session := sessions.Get(c)
	var name string
	user := models.User{}
	//var err error

	if nil != session.Get("loginname") {
		name = session.Get("loginname").(string)
		user, _ = userModel.GetUserByName(name)
	}
	tabs := [3]map[string]string{{"value": "share", "text": "分享"}, {"value": "ask", "text": "问答"}, {"value": "job", "text": "招聘"}}
	c.HTML(http.StatusOK, "edit", gin.H{
		"user": user,
		"tabs": tabs,
		"config": gin.H{
			"description": "CNode：Node.js专业中文社区",
		},
	})
}

func Index(c *gin.Context) {
	session := sessions.Get(c)
	var name string
	var no_reply_topics []models.Topic
	type Temp struct {
		Topic             models.Topic
		Author            models.User
		Replies           []models.Reply
		RepliyWithAuthors []models.ReplyAndAuthor
	}
	var temp Temp
	user := models.User{}
	if nil != session.Get("loginname") {
		name = session.Get("loginname").(string)
		user, _ = userModel.GetUserByName(name)
	}
	id := c.Param("id")
	topic, author, replies, repliyWithAuthors, _ := topicModel.GetTopicByIdWithReply(id)
	temp.Author = author
	temp.Topic = topic
	temp.Replies = replies
	NoOfRepliy := len(replies)
	temp.RepliyWithAuthors = repliyWithAuthors
	no_reply_topics2, err2 := cache.Get("no_reply_topics")
	json.Unmarshal(no_reply_topics2.([]byte), &no_reply_topics)
	log.Println("temp")
	log.Println(err2)
	//log.Println(temp)
	if err2 != nil {
		no_reply_topics, _ = topicModel.GetTopicNoReply()
		no_reply_topics_json, _ := json.Marshal(no_reply_topics)
		cache.SetEx("no_reply_topics", no_reply_topics_json)
	}
	other_topics, _ := topicModel.GetAuthorOtherTopics(author.Id.Hex(), id)
	c.HTML(http.StatusOK, "topicIndex", gin.H{
		"title":               "布局页面",
		"user":                user,
		"topic":               temp,
		"NoOfRepliy":          NoOfRepliy,
		"no_reply_topics":     no_reply_topics,
		"author_other_topics": other_topics,
		"config": gin.H{
			"description": "CNode：Node.js专业中文社区",
		},
	})
}
func Create(c *gin.Context) {
	session := sessions.Get(c)
	var name string
	user := models.User{}
	if nil != session.Get("loginname") {
		name = session.Get("loginname").(string)
		user, _ = userModel.GetUserByName(name)
	}
	id := user.Id.Hex()
	log.Println(id)
	tab := c.Request.FormValue("tab")
	title := c.Request.FormValue("title")
	content := c.Request.FormValue("content")
	topic, _ := topicModel.NewAndSave(title, tab, id, content)
	url := "/topic/" + topic.Id.Hex()
	c.Redirect(301, url)
}
func Add(c *gin.Context) {
	topic_id := c.Param("topic_id")
	r_content := c.Request.FormValue("r_content")
	user_id := c.Request.FormValue("user_id")
	if user_id==""{
		c.HTML(http.StatusOK, "notify", gin.H{
			"error": "您还没登录呢",
		})
		return	
	}

	topic,_:=topicModel.GetTopicById(topic_id)
	reply, _ := replyModel.NewAndSave(r_content, topic_id, user_id, "")
	topicModel.UpdateReplyCount(topic_id, reply.Id)
	log.Println(r_content)
	r, _ := regexp.Compile("@([a-z0-9]+)")
	usersArray:=r.FindAllString(r_content, -1)
	log.Println(len(usersArray))
	var users map[string]int
	users=make(map[string]int)
	if len(usersArray)>0 {
		for _,user:=range usersArray{
			user=user[1 : len(user)]
			users[user]=1
		}
		for userName:=range users{
			log.Println(userName)
			user,_:=userModel.GetUserByName(userName)
			log.Println(user)
			if user_id!=user.Id.Hex() {
				messageModel.SendAtMessage(user.Id.Hex(),user_id,topic_id,reply.Id)
			}	
		}
	}
	log.Println(users)
	if topic.Author_id.Hex()!=user_id {
		messageModel.SendReplyMessage(topic.Author_id.Hex(),user_id,topic_id,reply.Id)
	}
	url := "/topic/" + topic_id
	c.Redirect(301, url)
}
func Edit(c *gin.Context) {

	reply_id := c.Param("reply_id")
	t_content := c.Request.FormValue("t_content")
	user_id := c.Request.FormValue("user_id")
	reply, _ := replyModel.GetReplyById(reply_id)

	if reply.Author_id.Hex() != user_id {
		c.HTML(http.StatusOK, "notify", gin.H{
			"error": "你不能编辑此回复",
		})
		return
	}

	replyModel.Update(t_content, reply_id)

	url := "/topic/" + reply.Topic_id.Hex() + "#" + reply.Id.Hex()
	c.Redirect(301, url)
}
func Delete(c *gin.Context) {
	log.Print("delete3")
	reply_id := c.Param("reply_id")
	session := sessions.Get(c)
	var name string
	user := models.User{}
	if nil != session.Get("loginname") {
		name = session.Get("loginname").(string)
		user, _ = userModel.GetUserByName(name)
	}
	reply, _ := replyModel.GetReplyById(reply_id)

	var msg struct {
		Status string `json:"status"`
	}
	if reply.Author_id.Hex() != user.Id.Hex() {

		msg.Status = "failed"
		c.JSON(http.StatusOK, msg)
		return
	}

	replyModel.Delete(reply_id)

	msg.Status = "success"
	c.JSON(http.StatusOK, msg)
}
func ShowEdit(c *gin.Context) {
	reply_id := c.Param("reply_id")
	user_id := c.Request.FormValue("user_id")
	reply, err := replyModel.GetReplyById(reply_id)

	if err != nil {
		c.HTML(http.StatusOK, "notify", gin.H{
			"error": "评论不存在",
		})
		return
	}
	c.HTML(http.StatusOK, "reply/edit", gin.H{
		"reply_id": reply.Id.Hex(),
		"content":  reply.Content,
		"user_id":  user_id,
	})
}
