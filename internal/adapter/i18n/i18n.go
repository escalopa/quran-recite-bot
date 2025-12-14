package i18n

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/escalopa/quran-read-bot/internal/domain"
	"gopkg.in/yaml.v3"
)

type I18n struct {
	translations map[domain.Language]map[string]string
	surahs       map[domain.Language][]string
}

type translationFile struct {
	Messages map[string]string `yaml:"messages"`
	Surahs   []string          `yaml:"surahs"`
}

func NewI18n(localesDir string) (*I18n, error) {
	i18n := &I18n{
		translations: make(map[domain.Language]map[string]string),
		surahs:       make(map[domain.Language][]string),
	}

	// Load all translation files
	languages := []domain.Language{domain.LangEnglish, domain.LangArabic, domain.LangRussian}
	for _, lang := range languages {
		filename := filepath.Join(localesDir, string(lang)+".yaml")
		if err := i18n.loadTranslations(lang, filename); err != nil {
			return nil, fmt.Errorf("load %s translations: %w", lang, err)
		}
	}

	return i18n, nil
}

func (i *I18n) loadTranslations(lang domain.Language, filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	var tf translationFile
	if err := yaml.Unmarshal(data, &tf); err != nil {
		return fmt.Errorf("unmarshal yaml: %w", err)
	}

	i.translations[lang] = tf.Messages
	i.surahs[lang] = tf.Surahs

	return nil
}

// Get retrieves a translated message
func (i *I18n) Get(lang domain.Language, key string, args ...interface{}) string {
	translations, ok := i.translations[lang]
	if !ok {
		translations = i.translations[domain.LangEnglish]
	}

	msg, ok := translations[key]
	if !ok {
		return key
	}

	// Simple formatting support
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}

	return msg
}

// GetSurahName retrieves the localized name of a Surah
func (i *I18n) GetSurahName(lang domain.Language, surahNumber int) string {
	surahs, ok := i.surahs[lang]
	if !ok || surahNumber < 1 || surahNumber > len(surahs) {
		surahs = i.surahs[domain.LangEnglish]
	}

	if surahNumber < 1 || surahNumber > len(surahs) {
		return fmt.Sprintf("Surah %d", surahNumber)
	}

	return surahs[surahNumber-1]
}

// FormatSurahButton formats a surah button text with number and name
func FormatSurahButton(lang domain.Language, i18n *I18n, surahNumber int) string {
	name := i18n.GetSurahName(lang, surahNumber)
	return fmt.Sprintf("%d. %s", surahNumber, strings.TrimSpace(name))
}
