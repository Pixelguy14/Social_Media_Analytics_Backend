package i18n

import (
	"encoding/json"
	"os"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

var Bundle *i18n.Bundle

func Init_localization() error {
	Bundle = i18n.NewBundle(language.English)
	Bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	langs := []string{"en", "de", "fr", "es", "it", "ja", "hu", "fi", "pt", "nl"}
	for _, lang := range langs {
		_, err := Bundle.LoadMessageFile("i18n/locales/" + lang + ".json")
		if err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func Get_message(lang_tag string, message_id string) string {
	localizer := i18n.NewLocalizer(Bundle, lang_tag)
	msg, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID: message_id,
	})
	if err != nil {
		return ""
	}
	return msg
}
