package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	tg "github.com/semog/telegram-bot-api"
)

func handleCallbackQuery(bot *tg.BotAPI, update tg.Update, st Store) error {
	if update.CallbackQuery.Data == qryDummy {
		callbackConfig := tg.NewCallback(
			update.CallbackQuery.ID,
			"")
		_, err := bot.AnswerCallbackQuery(callbackConfig)
		if err != nil {
			return fmt.Errorf("could not send answer to callback query: %v", err)
		}

		return nil
	}

	if update.CallbackQuery.Data[0] == qryEditPayload {
		return handlePollEditQuery(bot, update, st)
	}

	if update.CallbackQuery.Data == qryCreatePoll {
		return sendNewQuestionMessage(bot, update, st)
	}

	if strings.Contains(update.CallbackQuery.Data, qryPollDone) {
		return handlePollDoneQuery(bot, update, st)
	}

	pollid, optionid, err := parseQueryPayload(update)
	if err != nil {
		return fmt.Errorf("could not parse query payload: %v", err)
	}

	if update.CallbackQuery.InlineMessageID != "" {
		if err := st.AddInlineMsgToPoll(pollid, update.CallbackQuery.InlineMessageID); err != nil {
			return fmt.Errorf("could not add inline message to poll: %v", err)
		}
	}

	p, err := st.GetPoll(pollid)
	if err != nil {
		return fmt.Errorf("could not get poll: %v", err)
	}
	if p.isInactive() {
		callbackConfig := tg.NewCallback(
			update.CallbackQuery.ID,
			"This poll is inactive.")
		_, err = bot.AnswerCallbackQuery(callbackConfig)
		if err != nil {
			return fmt.Errorf("could not send answer to callback query: %v", err)
		}
		return fmt.Errorf("poll %d is inactive", pollid)
	}

	newAnswer := answer{
		UserID:   update.CallbackQuery.From.ID,
		PollID:   pollid,
		OptionID: optionid,
	}
	unvoted, err := st.SaveAnswer(p, newAnswer)
	if err != nil {
		return fmt.Errorf("could not save answers: %v", err)
	}
	// polls were changed
	p, err = st.GetPoll(pollid)
	if err != nil {
		return fmt.Errorf("could not get poll: %v", err)
	}

	var choice option
	for _, o := range p.Options {
		if o.ID == newAnswer.OptionID {
			choice = o
		}
	}

	pollsToUpdate.enqueue(p.ID)

	var popupText string
	if unvoted {
		popupText = fmt.Sprintf("Selection removed")
	} else {
		popupText = fmt.Sprintf(`You selected "%s"`, choice.Text)
	}

	callbackConfig := tg.NewCallback(
		update.CallbackQuery.ID,
		popupText)
	_, err = bot.AnswerCallbackQuery(callbackConfig)
	if err != nil {
		return fmt.Errorf("could not send answer to callback query: %v", err)
	}

	return nil
}

func updatePollMessages(bot *tg.BotAPI, pollid int, st Store) error {
	p, err := st.GetPoll(pollid)
	if err != nil {
		return fmt.Errorf("could not get poll: %v", err)
	}

	//msgs, err := st.GetAllPollMsg(pollid)
	//if err != nil {
	//return fmt.Errorf("could not get all pollmsgs: %v", err)
	//}

	listing := buildPollListing(p, st)

	var ed tg.EditMessageTextConfig
	ed.Text = listing
	ed.ParseMode = tg.ModeHTML

	if !p.isInactive() {
		ed.ReplyMarkup = buildPollMarkup(p)
	}

	//for _, msg := range msgs {
	//ed.ChatID = msg.ChatID
	//ed.MessageID = msg.MessageID

	//_, err = bot.Send(ed)
	//if err != nil {
	//log.Printf("Could not edit message: %v \nThe message was: %s\n", err, ed.Text)
	//log.Printf("Could not edit message: %v\n", err)
	//splits := strings.Split(ed.Text, "\n")

	//ed.Text = ""
	//for _, l := range splits {
	//if !strings.HasPrefix(l, "\u251C") && !strings.HasPrefix(l, "\u2514") {
	//ed.Text += l + "\n"
	//}
	//}
	//log.Printf("try again:\n %s", ed.Text)
	//_, err = bot.Send(ed)
	//if err != nil {
	//log.Printf("could not update message: %v\n", err)
	//continue
	//}
	//}
	//}

	// reset
	ed.ChatID = 0
	ed.MessageID = 0

	msgs, err := st.GetAllPollInlineMsg(p.ID)
	if err != nil {
		return fmt.Errorf("could not get all poll inline messages: %v", err)
	}

	for _, msg := range msgs {
		ed.Text = listing
		ed.InlineMessageID = msg.InlineMessageID
		_, err := bot.Send(ed)
		if err != nil {
			if strings.Contains(err.Error(), "MESSAGE_ID_INVALID") {
				log.Printf("Remove inline message %s\n", msg.InlineMessageID)
				st.RemoveInlineMsg(msg.InlineMessageID)
				continue
			}
			// if strings.Contains(err.Error(), "chat not found") {
			// 	log.Printf("Remove inline message %s\n", msg.InlineMessageID)
			// 	st.RemoveInlineMsg(msg.InlineMessageID)
			// }
			if strings.Contains(err.Error(), "message is not modified") {
				continue
			}
			time.Sleep(20 * time.Millisecond)
			log.Printf("\n\n\nCould not edit inline message: %v \nThe message was: %s\n", err, ed.Text)
			log.Printf("Could not edit inline message: %v\n", err)
			splits := strings.Split(ed.Text, "\n")

			ed.Text = ""
			for _, l := range splits {
				if !strings.HasPrefix(l, "\u251C") && !strings.HasPrefix(l, "\u2514") {
					ed.Text += l + "\n"
				}
			}
			log.Printf("try again:\n %s", ed.Text)
			_, err = bot.Send(ed)
			if err != nil {
				log.Printf("could not update inline message: %v\n", err)
				continue
			}
		}
		time.Sleep(20 * time.Millisecond)
	}

	return nil
}

func handlePollDoneQuery(bot *tg.BotAPI, update tg.Update, st Store) error {
	splits := strings.Split(update.CallbackQuery.Data, ":")
	if len(splits) < 2 {
		return fmt.Errorf("query did not contain the pollid")
	}
	pollid, err := strconv.Atoi(splits[1])
	if err != nil {
		return fmt.Errorf("could not convert string payload to int: %v", err)
	}

	p, err := st.GetPoll(pollid)
	if err != nil {
		return fmt.Errorf("could not get poll: %v", err)
	}
	_, err = postPoll(bot, p, int64(update.CallbackQuery.From.ID))
	if err != nil {
		return fmt.Errorf("could not post finished poll: %v", err)
	}
	err = st.SaveState(update.CallbackQuery.From.ID, p.ID, pollDone)
	if err != nil {
		return fmt.Errorf("could not change state to poll done: %v", err)
	}
	return nil
}

func handlePollEditQuery(bot *tg.BotAPI, update tg.Update, st Store) error {
	splits := strings.Split(update.CallbackQuery.Data, ":")
	if len(splits) < 3 {
		log.Println(splits)
		return fmt.Errorf("query wrongly formatted")
	}
	pollid, err := strconv.Atoi(splits[1])
	if err != nil {
		return fmt.Errorf("could not convert string payload to int: %v", err)
	}

	var p *poll
	noNewer := false
	noOlder := false
	toggleInactive := false
	toggleMultipleChoice := false
	switch splits[2] {
	case qryNextPoll:
		p, err = st.GetPollNewer(pollid, update.CallbackQuery.From.ID)
		if err != nil {
			log.Printf("could not get older poll: %v\n", err)
			noNewer = true
		}
	case qryPrevPoll:
		p, err = st.GetPollOlder(pollid, update.CallbackQuery.From.ID)
		if err != nil {
			log.Printf("could not get older poll: %v\n", err)
			noOlder = true
		}
	case qryToggleActive:
		p, err = st.GetPoll(pollid)
		if err != nil {
			log.Printf("could not get poll: %v\n", err)
		}
		toggleInactive = true
	case qryToggleMultipleChoice:
		p, err = st.GetPoll(pollid)
		if err != nil {
			log.Printf("could not get poll: %v\n", err)
		}
		toggleMultipleChoice = true
	case qryAddOptions:
		state := waitingForOption
		err = st.SaveState(update.CallbackQuery.From.ID, pollid, state)
		if err != nil {
			return fmt.Errorf("could not save state: %v", err)
		}

		msg := tg.NewMessage(update.CallbackQuery.Message.Chat.ID, locAddOption)
		_, err = bot.Send(&msg)
		if err != nil {
			return fmt.Errorf("could not send message: %v", err)
		}
		return nil
	case qryEditQuestion:
		state := editQuestion
		err = st.SaveState(update.CallbackQuery.From.ID, pollid, state)
		if err != nil {
			return fmt.Errorf("could not save state: %v", err)
		}

		msg := tg.NewMessage(update.CallbackQuery.Message.Chat.ID, locEditQuestion)
		_, err = bot.Send(&msg)
		if err != nil {
			return fmt.Errorf("could not send message: %v", err)
		}
		return nil
	default:
		return fmt.Errorf("query wrongly formatted")
	}
	if err != nil {
		p, err = st.GetPoll(pollid)
		if err != nil {
			return fmt.Errorf("could not get poll by poll id: %v", err)
		}
	}

	// danger! malicious client could send pollid from another user in query
	if p.UserID != update.CallbackQuery.From.ID {
		return fmt.Errorf("user does not own poll: %v", err)
	}

	if toggleInactive {
		p.Inactive = 1 - p.Inactive // states are 0 and 1
		_, err = st.SavePoll(p)
		if err != nil {
			log.Println("Could not save toggled inactive state.")
		}
	}

	if toggleMultipleChoice {
		p.Type = 1 - p.Type // states are 0 and 1
		_, err = st.SavePoll(p)
		if err != nil {
			log.Println("Could not save toggled multiple choice state.")
		}
	}

	body := "Currently selected Poll:\n<pre>\n"
	body += p.Question + "\n" + lineSep + "\n"
	for i, o := range p.Options {
		body += fmt.Sprintf("%d. %s", i+1, o.Text) + "\n"
	}
	body += "</pre>\n\n"

	var ed tg.EditMessageTextConfig
	ed.Text = body
	ed.ParseMode = tg.ModeHTML
	ed.ReplyMarkup = buildEditMarkup(p, noOlder, noNewer)

	ed.ChatID = update.CallbackQuery.Message.Chat.ID
	ed.MessageID = update.CallbackQuery.Message.MessageID

	_, err = bot.Send(ed)
	if err != nil {
		return fmt.Errorf("could not update message: %v", err)
	}
	pollsToUpdate.enqueue(p.ID)
	return nil
}

func parseQueryPayload(update tg.Update) (pollid int, optionid int, err error) {
	dataSplit := strings.Split(update.CallbackQuery.Data, ":")
	if len(dataSplit) != 2 {
		return pollid, optionid, fmt.Errorf("could not parse response")
	}
	pollid, err = strconv.Atoi(dataSplit[0])
	if err != nil {
		return pollid, optionid, fmt.Errorf("could not convert CallbackQuery data pollid to int: %v", err)
	}

	optionid, err = strconv.Atoi(dataSplit[1])
	if err != nil {
		return pollid, optionid, fmt.Errorf("could not convert CallbackQuery data OptionID to int: %v", err)
	}
	return pollid, optionid, nil
}
