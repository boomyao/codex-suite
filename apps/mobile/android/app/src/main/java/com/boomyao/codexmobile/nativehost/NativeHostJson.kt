package com.boomyao.codexmobile.nativehost

import com.boomyao.codexmobile.shared.uniqueTrimmedStrings as sharedUniqueTrimmedStrings
import org.json.JSONArray
import org.json.JSONObject

fun jsonArrayStrings(values: JSONArray?): List<String> {
    if (values == null) {
        return emptyList()
    }
    val result = mutableListOf<String>()
    for (index in 0 until values.length()) {
        val value = values.optString(index).trim()
        if (value.isNotEmpty()) {
            result.add(value)
        }
    }
    return result
}

fun jsonObjectStringMap(value: JSONObject?): Map<String, String> {
    if (value == null) {
        return emptyMap()
    }
    val result = linkedMapOf<String, String>()
    val keys = value.keys()
    while (keys.hasNext()) {
        val key = keys.next()
        val text = value.optString(key).trim()
        if (text.isNotEmpty()) {
            result[key] = text
        }
    }
    return result
}

fun uniqueTrimmedStrings(values: Iterable<String>): MutableList<String> = sharedUniqueTrimmedStrings(values)

fun deepCopyJsonObject(value: JSONObject): JSONObject = JSONObject(value.toString())

fun deepCopyJsonValue(value: Any?): Any? =
    when (value) {
        null, JSONObject.NULL -> JSONObject.NULL
        is JSONObject -> JSONObject(value.toString())
        is JSONArray -> JSONArray(value.toString())
        is Map<*, *> -> {
            val objectValue = JSONObject()
            value.forEach { (key, entryValue) ->
                if (key is String) {
                    objectValue.put(key, deepCopyJsonValue(entryValue))
                }
            }
            objectValue
        }
        is Iterable<*> -> {
            val arrayValue = JSONArray()
            value.forEach { entryValue ->
                arrayValue.put(deepCopyJsonValue(entryValue))
            }
            arrayValue
        }
        else -> value
    }

fun toJsonCompatible(value: Any?): Any {
    return deepCopyJsonValue(value) ?: JSONObject.NULL
}
