package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/Rhymen/go-whatsapp"
	"github.com/Rhymen/go-whatsapp/binary/proto"
)

// Handler whatsapp connection handler
type Handler struct {
	wac       *whatsapp.Conn
	startTime time.Time
}

// HandleError handles an error
func (h Handler) HandleError(err error) {
	if e, ok := err.(*whatsapp.ErrConnectionFailed); ok {
		log.Printf("connection failed, underlying error: %v\n", e.Err)
		log.Println("waiting for 30 seconds ...")
		<-time.After(30 * time.Second)
		log.Println("reconnecting ...")
		err := h.wac.Restore()
		if err != nil {
			log.Fatalf("restore failed: %v", err)
		}
	} else {
		log.Printf("error occoured: %v\n", err)
	}
}

// HandleTextMessage handles a text message
func (h Handler) HandleTextMessage(message whatsapp.TextMessage) {
	if message.Info.Timestamp < uint64(h.startTime.Unix()) ||
		message.Info.Timestamp < uint64(time.Now().Unix()-30) ||
		message.Info.FromMe || message.Info.RemoteJid == "status@broadcast" {
		return
	}
	addSenderJid(&message)
	log.Printf("%v %v", message.Info.RemoteJid, message.Text)
	replyMessage := getTextReply(h, &message)
	if len(replyMessage) == 0 {
		return
	}
	text := whatsapp.TextMessage{
		Info: whatsapp.MessageInfo{
			RemoteJid: message.Info.RemoteJid,
		},
		ContextInfo: whatsapp.ContextInfo{
			QuotedMessage: &proto.Message{
				Conversation: &message.Text,
			},
			QuotedMessageID: message.Info.Id,
			Participant:     message.Info.SenderJid,
		},
		Text: replyMessage,
	}
	h.wac.Send(text)
}

func getTextReply(h Handler, message *whatsapp.TextMessage) string {
	if !isGroupChat(message) {
		return cnf.getMessageTemplate("private_chat")
	} else if cond, _ := (&groupModel{JID: message.Info.RemoteJid}).isExist(); !cond {
		if message.Text == "@register" {
			groupName := h.wac.Store.Contacts[message.Info.RemoteJid].Name
			checkError((&groupModel{message.Info.RemoteJid, groupName}).add())
			return fmt.Sprintf(cnf.getMessageTemplate("register_success"), groupName)
		}
		return cnf.getMessageTemplate("not_registered")
	} else if message.Text == "@unregister" {
		checkError((&groupModel{JID: message.Info.RemoteJid}).delete())
		return cnf.getMessageTemplate("unregister")
	} else {
		switch {
		case message.Text == "@menu":
			return cnf.getMessageTemplate("menu")
		case strings.HasPrefix(message.Text, "@tambah"):
			args := strings.SplitN(message.Text, " ", 3)
			if message.ContextInfo.QuotedMessageID == "" {
				return cnf.getMessageTemplate("no_assignment_description")
			} else if len(args) != 3 {
				return cnf.getMessageTemplate("invalid_add_assignment_args")
			} else if len(args[2]) > 30 || len(args[1]) > 10 {
				return cnf.getMessageTemplate("assignment_too_long")
			}
			checkError((&assignmentModel{
				GroupJID:    message.Info.RemoteJid,
				Subject:     args[1],
				Description: message.ContextInfo.QuotedMessage.GetConversation(),
				Deadline:    args[2],
			}).add())
			return fmt.Sprintf(cnf.getMessageTemplate("assignment_added"))
		case strings.HasPrefix(message.Text, "@delete"):
			args := strings.Split(message.Text, " ")
			if len(args) != 2 {
				return cnf.getMessageTemplate("invalid_args")
			}
			assignmentID, err := strconv.Atoi(args[1])
			assignment := &assignmentModel{ID: assignmentID}
			if err != nil {
				return cnf.getMessageTemplate("invalid_args")
			} else if cond, _ := assignment.isExist(); !cond {
				return cnf.getMessageTemplate("invalid_assignment_id")
			}
			checkError(assignment.delete())
			return cnf.getMessageTemplate("assignment_deleted")
		default:
			return ""
		}
	}
}

func isGroupChat(message *whatsapp.TextMessage) bool {
	return strings.HasSuffix(message.Info.RemoteJid, "g.us")
}

func addSenderJid(message *whatsapp.TextMessage) {
	message.Info.SenderJid = message.Info.RemoteJid
	if len(message.Info.Source.GetParticipant()) != 0 {
		message.Info.SenderJid = message.Info.Source.GetParticipant()
	}
}
