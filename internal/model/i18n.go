package model

func ReasonLabel(code ReasonCode, locale string) string {
	labels, ok := reasonLabels[code]
	if !ok {
		return string(code)
	}
	if locale == "en" {
		return labels.en
	}
	return labels.ru
}

type bilingual struct {
	ru, en string
}

var reasonLabels = map[ReasonCode]bilingual{
	ReasonNoProvider:            {"Нет провайдера для сегмента", "No provider available for leg"},
	ReasonNoOffers:              {"Нет предложений", "No offers returned"},
	ReasonConnectionTooShort:    {"Слишком короткая стыковка", "Connection too short"},
	ReasonConnectionTooLong:     {"Слишком длинная стыковка", "Connection too long"},
	ReasonTransitVisaRequired:   {"Требуется транзитная виза", "Transit visa required"},
	ReasonBaggageUnknown:        {"Неизвестны условия багажа", "Baggage information unknown"},
	ReasonAPIError:              {"Ошибка API провайдера", "Provider API error"},
	ReasonCurrencyUnconvertible: {"Валюта не конвертируется", "Currency cannot be converted"},
}

func AllReasonCodes() []ReasonCode {
	return []ReasonCode{
		ReasonNoProvider,
		ReasonNoOffers,
		ReasonConnectionTooShort,
		ReasonConnectionTooLong,
		ReasonTransitVisaRequired,
		ReasonBaggageUnknown,
		ReasonAPIError,
		ReasonCurrencyUnconvertible,
	}
}
