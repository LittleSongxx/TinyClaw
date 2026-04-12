package i18n

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/LittleSongxx/TinyClaw/conf"
	"github.com/LittleSongxx/TinyClaw/logger"
	botUtils "github.com/LittleSongxx/TinyClaw/utils"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

var (
	ruLocalizer *i18n.Localizer
	enLocalizer *i18n.Localizer
	zhLocalizer *i18n.Localizer
)

const (
	ru = "ru"
	en = "en"
	zh = "zh"
)

func InitI18n() {
	// 1. Create a new i18n bundle with English as default language
	bundle := i18n.NewBundle(language.English)

	// 2. Register JSON unmarshal function (other formats like TOML are also supported)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	// 3. Load translation files
	loadTranslationFile(bundle, "conf/i18n/i18n.ru.json", "Russian", false)
	loadTranslationFile(bundle, "conf/i18n/i18n.en.json", "English", true)
	loadTranslationFile(bundle, "conf/i18n/i18n.zh.json", "Chinese", true)

	// 4. Create localizers for each language
	ruLocalizer = i18n.NewLocalizer(bundle, ru)
	enLocalizer = i18n.NewLocalizer(bundle, en)
	zhLocalizer = i18n.NewLocalizer(bundle, zh)
}

func loadTranslationFile(bundle *i18n.Bundle, relPath, languageName string, required bool) {
	if bundle == nil {
		return
	}

	absPath := botUtils.GetAbsPath(relPath)
	_, err := bundle.LoadMessageFile(absPath)
	if err == nil {
		return
	}

	if !required && errors.Is(err, os.ErrNotExist) {
		return
	}

	if !required {
		logger.Warn("Failed to load optional translation file", "language", languageName, "path", absPath, "err", err)
		return
	}

	logger.Error("Failed to load translation file", "language", languageName, "path", absPath, "err", err)
}

// GetMessage function to get localized message
func GetMessage(messageID string, templateData map[string]interface{}) string {
	var localizer *i18n.Localizer
	switch conf.BaseConfInfo.Lang {
	case ru:
		localizer = ruLocalizer
	case zh:
		localizer = zhLocalizer
	default:
		localizer = enLocalizer
	}

	msg, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID:    messageID,
		TemplateData: templateData,
	})
	if err != nil {
		logger.Warn("Failed to localize message", "tag", conf.BaseConfInfo.Lang, "messageID", messageID, "err", err)
		return ""
	}
	return msg
}
