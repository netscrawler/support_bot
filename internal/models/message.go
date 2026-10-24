package models

type Message struct {
	ReportName string

	Meta map[string]any

	Recipients []Recipient

	Text     []Data
	RichText []Data
	Files    []Data
	Images   []Data
}

func NewMessage(
	rName string,
	data []Data,
	additionalInfo map[string]any,
	rcpts ...Recipient,
) *Message {
	var txt, rTxt, fl, imgs []Data

	for _, d := range data {
		switch d.Type {
		case SendTextKind:
			txt = append(txt, d)
		case SendRichTextKind:
			rTxt = append(rTxt, d)
		case sendFileKind:
			fl = append(fl, d)
		case sendImageKind:
			imgs = append(imgs, d)
		default:
		}
	}

	return &Message{
		ReportName: rName,
		Recipients: rcpts,
		Text:       txt,
		RichText:   rTxt,
		Files:      fl,
		Images:     imgs,
		Meta:       additionalInfo,
	}
}
