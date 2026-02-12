package tui

// layoutMap maps non-English keyboard runes to their English (US QWERTY)
// equivalents based on physical key position. This allows shortcuts like
// j/k navigation to work when a non-Latin keyboard layout is active.
//
// Covered layouts: Russian (ЙЦУКЕН), Ukrainian, Hebrew, Arabic.
var layoutMap = map[rune]rune{
	// Russian lowercase (ЙЦУКЕН layout)
	'й': 'q', 'ц': 'w', 'у': 'e', 'к': 'r', 'е': 't',
	'н': 'y', 'г': 'u', 'ш': 'i', 'щ': 'o', 'з': 'p',
	'ф': 'a', 'ы': 's', 'в': 'd', 'а': 'f', 'п': 'g',
	'р': 'h', 'о': 'j', 'л': 'k', 'д': 'l',
	'я': 'z', 'ч': 'x', 'с': 'c', 'м': 'v', 'и': 'b',
	'т': 'n', 'ь': 'm',

	// Russian uppercase
	'Й': 'Q', 'Ц': 'W', 'У': 'E', 'К': 'R', 'Е': 'T',
	'Н': 'Y', 'Г': 'U', 'Ш': 'I', 'Щ': 'O', 'З': 'P',
	'Ф': 'A', 'Ы': 'S', 'В': 'D', 'А': 'F', 'П': 'G',
	'Р': 'H', 'О': 'J', 'Л': 'K', 'Д': 'L',
	'Я': 'Z', 'Ч': 'X', 'С': 'C', 'М': 'V', 'И': 'B',
	'Т': 'N', 'Ь': 'M',

	// Ukrainian-specific (keys that differ from Russian)
	'і': 's', 'І': 'S', // s key (Russian has ы/Ы)
	'є': '\'', 'Є': '"', // ' key (Russian has э/Э)
	'ї': ']', 'Ї': '}', // ] key (Russian has ъ/Ъ)
	'ґ': '`', 'Ґ': '~', // ` key (Russian has ё/Ё)

	// Hebrew (no case distinction)
	'ק': 'e', 'ר': 'r', 'א': 't', 'ט': 'y', 'ו': 'u',
	'ן': 'i', 'ם': 'o', 'פ': 'p',
	'ש': 'a', 'ד': 's', 'ג': 'd', 'כ': 'f', 'ע': 'g',
	'י': 'h', 'ח': 'j', 'ל': 'k', 'ך': 'l',
	'ז': 'z', 'ס': 'x', 'ב': 'c', 'ה': 'v', 'נ': 'b',
	'מ': 'n', 'צ': 'm',

	// Arabic (standard Arabic 101 layout)
	'ض': 'q', 'ص': 'w', 'ث': 'e', 'ق': 'r', 'ف': 't',
	'غ': 'y', 'ع': 'u', 'ه': 'i', 'خ': 'o', 'ح': 'p',
	'ش': 'a', 'س': 's', 'ي': 'd', 'ب': 'f', 'ل': 'g',
	'ا': 'h', 'ت': 'j', 'ن': 'k', 'م': 'l',
	'ئ': 'z', 'ء': 'x', 'ؤ': 'c', 'ر': 'v',
	'ى': 'n', 'ة': 'm',
}

// TranslateRune maps a non-English keyboard rune to its English equivalent
// based on physical key position. Returns the original rune if no mapping exists.
func TranslateRune(r rune) rune {
	if mapped, ok := layoutMap[r]; ok {
		return mapped
	}
	return r
}
