#include <jni.h>
#include <android/log.h>
#include <dlfcn.h>
#include <pthread.h>
#include <stdlib.h>
#include <stdio.h>

typedef char *(*bridge_noarg_fn)(void);
typedef char *(*bridge_arg2_int_fn)(const char *, const char *, int);
typedef char *(*bridge_arg2_fn)(const char *, const char *);
typedef void (*bridge_free_fn)(char *);
typedef void (*bridge_set_route_fn)(const char *);
typedef void (*bridge_set_snapshot_fn)(const char *);

static void *bridge_handle = NULL;
static bridge_noarg_fn bridge_version = NULL;
static bridge_noarg_fn bridge_status = NULL;
static bridge_arg2_int_fn bridge_start = NULL;
static bridge_arg2_fn bridge_configure_proxy = NULL;
static bridge_noarg_fn bridge_stop = NULL;
static bridge_free_fn bridge_free = NULL;
static bridge_set_route_fn bridge_set_default_route = NULL;
static bridge_set_snapshot_fn bridge_set_interface_snapshot = NULL;
static pthread_mutex_t bridge_mutex = PTHREAD_MUTEX_INITIALIZER;
static JavaVM *bridge_vm = NULL;
static jobject vpn_service = NULL;
static jmethodID vpn_protect_method = NULL;
static pthread_mutex_t vpn_service_mutex = PTHREAD_MUTEX_INITIALIZER;

static void write_error(char *errbuf, size_t errlen, const char *message) {
    if (errbuf == NULL || errlen == 0) {
        return;
    }
    snprintf(errbuf, errlen, "%s", message != NULL ? message : "unknown error");
}

static int ensure_bridge_loaded(void) {
    if (bridge_handle != NULL && bridge_version != NULL && bridge_status != NULL &&
        bridge_start != NULL && bridge_configure_proxy != NULL && bridge_stop != NULL && bridge_free != NULL &&
        bridge_set_default_route != NULL && bridge_set_interface_snapshot != NULL) {
        return 1;
    }

    pthread_mutex_lock(&bridge_mutex);
    if (bridge_handle != NULL && bridge_version != NULL && bridge_status != NULL &&
        bridge_start != NULL && bridge_configure_proxy != NULL && bridge_stop != NULL && bridge_free != NULL &&
        bridge_set_default_route != NULL && bridge_set_interface_snapshot != NULL) {
        pthread_mutex_unlock(&bridge_mutex);
        return 1;
    }

    void *handle = dlopen("libcodexmobile.so", RTLD_NOW | RTLD_GLOBAL);
    if (handle == NULL) {
        __android_log_print(ANDROID_LOG_ERROR, "codexmobile_jni", "dlopen failed: %s", dlerror());
        pthread_mutex_unlock(&bridge_mutex);
        return 0;
    }

    bridge_noarg_fn version = (bridge_noarg_fn) dlsym(handle, "CodexMobileVersionJSON");
    bridge_noarg_fn status = (bridge_noarg_fn) dlsym(handle, "CodexMobileStatusJSON");
    bridge_arg2_int_fn start = (bridge_arg2_int_fn) dlsym(handle, "CodexMobileStartWithTunFDJSON");
    bridge_arg2_fn configure_proxy = (bridge_arg2_fn) dlsym(handle, "CodexMobileConfigureBridgeProxyJSON");
    bridge_noarg_fn stop = (bridge_noarg_fn) dlsym(handle, "CodexMobileStopJSON");
    bridge_free_fn free_fn = (bridge_free_fn) dlsym(handle, "CodexMobileFreeString");
    bridge_set_route_fn set_default_route = (bridge_set_route_fn) dlsym(handle, "CodexMobileSetAndroidDefaultRouteInterface");
    bridge_set_snapshot_fn set_interface_snapshot = (bridge_set_snapshot_fn) dlsym(handle, "CodexMobileSetAndroidInterfaceSnapshot");
    if (version == NULL || status == NULL || start == NULL || configure_proxy == NULL || stop == NULL || free_fn == NULL ||
        set_default_route == NULL || set_interface_snapshot == NULL) {
        __android_log_print(
                ANDROID_LOG_ERROR,
                "codexmobile_jni",
                "dlsym failed: version=%p status=%p start=%p configure_proxy=%p stop=%p free=%p setRoute=%p setSnapshot=%p",
                version,
                status,
                start,
                configure_proxy,
                stop,
                free_fn,
                set_default_route,
                set_interface_snapshot
        );
        dlclose(handle);
        pthread_mutex_unlock(&bridge_mutex);
        return 0;
    }

    bridge_handle = handle;
    bridge_version = version;
    bridge_status = status;
    bridge_start = start;
    bridge_configure_proxy = configure_proxy;
    bridge_stop = stop;
    bridge_free = free_fn;
    bridge_set_default_route = set_default_route;
    bridge_set_interface_snapshot = set_interface_snapshot;
    pthread_mutex_unlock(&bridge_mutex);
    return 1;
}

static jstring invoke_noarg(JNIEnv *env, bridge_noarg_fn callback) {
    if (!ensure_bridge_loaded() || callback == NULL) {
        return (*env)->NewStringUTF(env, "{\"ok\":false,\"error\":\"bridge load failed\"}");
    }

    char *result = callback();
    jstring output = (*env)->NewStringUTF(
            env,
            result != NULL ? result : "{\"ok\":false,\"error\":\"empty bridge response\"}"
    );
    if (result != NULL) {
        bridge_free(result);
    }
    return output;
}

static jstring invoke_with_two_strings_and_int(JNIEnv *env, jstring first, jstring second, jint value) {
    if (!ensure_bridge_loaded()) {
        return (*env)->NewStringUTF(env, "{\"ok\":false,\"error\":\"bridge load failed\"}");
    }

    const char *left = "";
    const char *right = "";
    if (first != NULL) {
        left = (*env)->GetStringUTFChars(env, first, NULL);
    }
    if (second != NULL) {
        right = (*env)->GetStringUTFChars(env, second, NULL);
    }

    char *result = bridge_start(left, right, value);

    if (second != NULL) {
        (*env)->ReleaseStringUTFChars(env, second, right);
    }
    if (first != NULL) {
        (*env)->ReleaseStringUTFChars(env, first, left);
    }

    jstring output = (*env)->NewStringUTF(
            env,
            result != NULL ? result : "{\"ok\":false,\"error\":\"empty bridge response\"}"
    );
    if (result != NULL) {
        bridge_free(result);
    }
    return output;
}

static jstring invoke_with_two_strings(JNIEnv *env, bridge_arg2_fn callback, jstring first, jstring second) {
    if (!ensure_bridge_loaded() || callback == NULL) {
        return (*env)->NewStringUTF(env, "{\"ok\":false,\"error\":\"bridge load failed\"}");
    }

    const char *left = "";
    const char *right = "";
    if (first != NULL) {
        left = (*env)->GetStringUTFChars(env, first, NULL);
    }
    if (second != NULL) {
        right = (*env)->GetStringUTFChars(env, second, NULL);
    }

    char *result = callback(left, right);

    if (second != NULL) {
        (*env)->ReleaseStringUTFChars(env, second, right);
    }
    if (first != NULL) {
        (*env)->ReleaseStringUTFChars(env, first, left);
    }

    jstring output = (*env)->NewStringUTF(
            env,
            result != NULL ? result : "{\"ok\":false,\"error\":\"empty bridge response\"}"
    );
    if (result != NULL) {
        bridge_free(result);
    }
    return output;
}

static void clear_vpn_service_locked(JNIEnv *env) {
    if (vpn_service != NULL) {
        (*env)->DeleteGlobalRef(env, vpn_service);
        vpn_service = NULL;
    }
    vpn_protect_method = NULL;
}

static int set_vpn_service_locked(JNIEnv *env, jobject service) {
    clear_vpn_service_locked(env);
    if (service == NULL) {
        return 1;
    }

    jobject global_service = (*env)->NewGlobalRef(env, service);
    if (global_service == NULL) {
        return 0;
    }

    jclass service_class = (*env)->GetObjectClass(env, service);
    if (service_class == NULL) {
        (*env)->DeleteGlobalRef(env, global_service);
        return 0;
    }

    jmethodID protect_method = (*env)->GetMethodID(env, service_class, "protect", "(I)Z");
    (*env)->DeleteLocalRef(env, service_class);
    if (protect_method == NULL) {
        (*env)->DeleteGlobalRef(env, global_service);
        return 0;
    }

    vpn_service = global_service;
    vpn_protect_method = protect_method;
    return 1;
}

jint JNI_OnLoad(JavaVM *vm, void *reserved) {
    (void) reserved;
    setenv("TS_NO_LOGS_NO_SUPPORT", "true", 1);
    bridge_vm = vm;
    return JNI_VERSION_1_6;
}

JNIEXPORT jstring JNICALL
Java_com_boomyao_codexmobile_tailnet_CodexTailnetNative_version(JNIEnv *env, jclass clazz) {
    (void) clazz;
    return invoke_noarg(env, bridge_version);
}

JNIEXPORT jstring JNICALL
Java_com_boomyao_codexmobile_tailnet_CodexTailnetNative_status(JNIEnv *env, jclass clazz) {
    (void) clazz;
    return invoke_noarg(env, bridge_status);
}

JNIEXPORT jstring JNICALL
Java_com_boomyao_codexmobile_tailnet_CodexTailnetNative_start(JNIEnv *env, jclass clazz, jstring payload_json, jstring state_dir, jint tun_fd) {
    (void) clazz;
    return invoke_with_two_strings_and_int(env, payload_json, state_dir, tun_fd);
}

JNIEXPORT jstring JNICALL
Java_com_boomyao_codexmobile_tailnet_CodexTailnetNative_configureBridgeProxy(JNIEnv *env, jclass clazz, jstring endpoint, jstring auth_token) {
    (void) clazz;
    return invoke_with_two_strings(env, bridge_configure_proxy, endpoint, auth_token);
}

JNIEXPORT jstring JNICALL
Java_com_boomyao_codexmobile_tailnet_CodexTailnetNative_stop(JNIEnv *env, jclass clazz) {
    (void) clazz;
    return invoke_noarg(env, bridge_stop);
}

JNIEXPORT void JNICALL
Java_com_boomyao_codexmobile_tailnet_CodexTailnetNative_installVpnService(JNIEnv *env, jclass clazz, jobject service) {
    (void) clazz;
    pthread_mutex_lock(&vpn_service_mutex);
    set_vpn_service_locked(env, service);
    pthread_mutex_unlock(&vpn_service_mutex);
}

JNIEXPORT void JNICALL
Java_com_boomyao_codexmobile_tailnet_CodexTailnetNative_clearVpnService(JNIEnv *env, jclass clazz) {
    (void) clazz;
    pthread_mutex_lock(&vpn_service_mutex);
    clear_vpn_service_locked(env);
    pthread_mutex_unlock(&vpn_service_mutex);
}

JNIEXPORT void JNICALL
Java_com_boomyao_codexmobile_tailnet_CodexTailnetNative_setDefaultRouteInterface(JNIEnv *env, jclass clazz, jstring interface_name) {
    (void) clazz;
    if (!ensure_bridge_loaded() || bridge_set_default_route == NULL) {
        return;
    }
    const char *raw = NULL;
    if (interface_name != NULL) {
        raw = (*env)->GetStringUTFChars(env, interface_name, NULL);
    }
    bridge_set_default_route(raw);
    if (interface_name != NULL && raw != NULL) {
        (*env)->ReleaseStringUTFChars(env, interface_name, raw);
    }
}

JNIEXPORT void JNICALL
Java_com_boomyao_codexmobile_tailnet_CodexTailnetNative_setInterfaceSnapshot(JNIEnv *env, jclass clazz, jstring snapshot_json) {
    (void) clazz;
    if (!ensure_bridge_loaded() || bridge_set_interface_snapshot == NULL) {
        return;
    }
    const char *raw = NULL;
    if (snapshot_json != NULL) {
        raw = (*env)->GetStringUTFChars(env, snapshot_json, NULL);
    }
    bridge_set_interface_snapshot(raw);
    if (snapshot_json != NULL && raw != NULL) {
        (*env)->ReleaseStringUTFChars(env, snapshot_json, raw);
    }
}

JNIEXPORT int JNICALL
CodexMobileProtectSocketFD(int fd, char *errbuf, size_t errlen) {
    if (bridge_vm == NULL) {
        write_error(errbuf, errlen, "JavaVM unavailable");
        return -1;
    }

    JNIEnv *env = NULL;
    jint env_status = (*bridge_vm)->GetEnv(bridge_vm, (void **) &env, JNI_VERSION_1_6);
    int should_detach = 0;
    if (env_status == JNI_EDETACHED) {
        if ((*bridge_vm)->AttachCurrentThread(bridge_vm, &env, NULL) != JNI_OK) {
            write_error(errbuf, errlen, "AttachCurrentThread failed");
            return -1;
        }
        should_detach = 1;
    } else if (env_status != JNI_OK) {
        write_error(errbuf, errlen, "GetEnv failed");
        return -1;
    }

    jobject service = NULL;
    jmethodID protect_method = NULL;

    pthread_mutex_lock(&vpn_service_mutex);
    if (vpn_service != NULL) {
        service = (*env)->NewLocalRef(env, vpn_service);
        protect_method = vpn_protect_method;
    }
    pthread_mutex_unlock(&vpn_service_mutex);

    if (service == NULL || protect_method == NULL) {
        if (service != NULL) {
            (*env)->DeleteLocalRef(env, service);
        }
        if (should_detach) {
            (*bridge_vm)->DetachCurrentThread(bridge_vm);
        }
        return 0;
    }

    jboolean allowed = (*env)->CallBooleanMethod(env, service, protect_method, fd);
    int has_exception = (*env)->ExceptionCheck(env);
    if (has_exception) {
        (*env)->ExceptionDescribe(env);
        (*env)->ExceptionClear(env);
    }
    (*env)->DeleteLocalRef(env, service);

    if (should_detach) {
        (*bridge_vm)->DetachCurrentThread(bridge_vm);
    }

    if (has_exception) {
        write_error(errbuf, errlen, "VpnService.protect failed");
        return -1;
    }
    if (allowed != JNI_TRUE) {
        __android_log_print(
                ANDROID_LOG_WARN,
                "codexmobile_jni",
                "VpnService.protect(%d) returned false; treating as no-op because no full VPN tunnel is established yet",
                fd
        );
        return 0;
    }

    return 0;
}
