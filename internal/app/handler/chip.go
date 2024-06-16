package handler

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/damonto/telegram-sms/internal/pkg/lpac"
	"github.com/damonto/telegram-sms/internal/pkg/util"
	"gopkg.in/telebot.v3"
)

type ChipHandler struct {
	handler
}

func HandleChipCommand(c telebot.Context) error {
	h := &ChipHandler{}
	h.setModem(c)
	return h.Handle(c)
}

func (h *ChipHandler) Handle(c telebot.Context) error {
	h.modem.Lock()
	defer h.modem.Unlock()
	usbDevice, err := h.modem.GetAtPort()
	if err != nil {
		return err
	}
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	chip, err := lpac.NewCmd(timeoutCtx, usbDevice).Info()
	if err != nil {
		return err
	}

	message := `
EID: %s
Manufaturer: %s
Free Space: %d KiB
Sign Keys:
%s
`
	country, manufacturer, productName := util.MatchEUM(chip.EID)
	var manufacturerReplacement string
	if country != "" {
		manufacturerReplacement += string(0x1F1E6+rune(country[0])-'A') + string(0x1F1E6+rune(country[1])-'A')
	}
	if manufacturer != "" {
		manufacturerReplacement += " " + manufacturer
	}
	if productName != "" {
		manufacturerReplacement += " " + productName
	}
	manufacturerReplacement += " " + chip.EUICCInfo2.SasAccreditationNumber
	manufacturerReplacement = strings.TrimRight(strings.TrimLeft(manufacturerReplacement, " "), " ")

	var keysReplacement string
	for _, key := range chip.EUICCInfo2.PkiForSigning {
		keysReplacement += util.FincCertificateIssuer(key) + "\n"
	}
	keysReplacement = strings.TrimSuffix(keysReplacement, "\n")

	return c.Send(
		util.EscapeText(fmt.Sprintf(
			message,
			chip.EID,
			manufacturerReplacement,
			chip.EUICCInfo2.ExtCardResource.FreeNonVolatileMemory/1024,
			keysReplacement,
		)),
		&telebot.SendOptions{
			ParseMode: telebot.ModeMarkdownV2,
		},
	)
}