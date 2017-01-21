/*
 * The reason why this addon is needed is that I need to determine
 * which socket can be connected to before I perform any operations,
 * otherwise I' ll get a runtime error, and the APIs provided by node.js
 * are all asynchronous, I cannot export a docker client which I am sure
 * that it has connected to a being listened socket.
 *
 * I perfer connecting to TCP socket in development so that I can
 * capture the packets to inspect the json that is passed to the docker daemon.
 *
 * As a result, I decided to check TCP socket first and fallback to the default
 * UNIX local socket in deveplopment environment.
 */

#include <cstdlib>
#include <cstring>
#include <string>

#include <unistd.h>
#include <sys/un.h>
#include <arpa/inet.h>
#include <netdb.h>

#include <nan.h>

namespace demo {

using v8::FunctionTemplate;
using v8::String;

using std::string;

typedef uint32_t u32;
typedef uint16_t u16;
typedef int32_t i32;

bool checkSocket(const string &fp) {
    struct sockaddr_un addr;
    int fd;
    if ( (fd = socket(AF_UNIX, SOCK_STREAM, 0)) < 0) {
        return false;
    }
    bzero((char *) &addr, sizeof(addr));
    addr.sun_family = AF_UNIX;
    strncpy(addr.sun_path, fp.c_str(), sizeof(addr.sun_path)-1);
    if (connect(fd, (struct sockaddr*)&addr, sizeof(addr)) == -1) {
        close(fd);
        return false;
    }
    close(fd);
    return true;
}

bool checkSocket(const string &host, u16 port) {
    struct sockaddr_in addr;
    int fd;
    if ( (fd = socket(AF_INET, SOCK_STREAM, 0)) < 0) {
        return false;
    }
    bzero((char *) &addr, sizeof(addr));
    addr.sin_family = AF_INET;
    addr.sin_port = htons(port);
    in_addr_t ip = inet_addr(host.c_str());
    if (ip == (in_addr_t) -1) {
        struct hostent *server;
        server = gethostbyname(host.c_str());
        if (server == NULL) {
            return false;
        }
        bcopy((char *)server->h_addr, (char *)&ip, (size_t)server->h_length);
    }
    addr.sin_addr.s_addr = ip;
    if (connect(fd, (struct sockaddr*)&addr, sizeof(addr)) == -1) {
        close(fd);
        return false;
    }
    close(fd);
    return true;
}

NAN_METHOD(IsListening)
{
// Usage:
// 1. isListening(socketPath: string) -> bool
// 2. isListening(address: string, port: u16) -> bool
    bool ret = false;
    i32 len = info.Length();
    if (len == 1) {
        string fp(*String::Utf8Value(info[0]->ToString()));
        ret = checkSocket(fp);
    } else if (len == 2) {
        string addr(*String::Utf8Value(info[0]->ToString()));
        u16 port = (u16)info[1]->Uint32Value();
        ret = checkSocket(addr, port);
    } else {
        return Nan::ThrowError("Wrong number of arguments");
    }
    info.GetReturnValue().Set(ret);
}

NAN_MODULE_INIT(Init)
{
  Nan::Set(target, Nan::New("isListening").ToLocalChecked(),
           Nan::GetFunction(Nan::New<FunctionTemplate>(IsListening)).ToLocalChecked());
}

NODE_MODULE(addon, Init);
}
