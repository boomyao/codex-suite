cmake_minimum_required(VERSION 3.22.1)

project(codexmobile_jni C)

add_library(codexmobile_jni SHARED codexmobile_jni.c)

find_library(log-lib log)

target_link_libraries(codexmobile_jni ${log-lib} dl)
