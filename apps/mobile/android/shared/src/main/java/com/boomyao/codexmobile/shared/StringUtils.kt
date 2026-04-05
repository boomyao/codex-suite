package com.boomyao.codexmobile.shared

fun uniqueTrimmedStrings(values: Iterable<String>): MutableList<String> {
    val result = mutableListOf<String>()
    val seen = linkedSetOf<String>()
    values.forEach { value ->
        val normalized = value.trim()
        if (normalized.isNotEmpty() && seen.add(normalized)) {
            result.add(normalized)
        }
    }
    return result
}
