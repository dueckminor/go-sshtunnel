#include <stdio.h>
#include <syslog.h>
#include <sys/socket.h>
#include <sys/time.h>
#include <sys/types.h>
#include <unistd.h>
#include <sys/poll.h>
#include <dlfcn.h>
#include <arpa/inet.h>
#include <netdb.h>
#include <stdlib.h>

static bool s_bActive = false;

__attribute__((constructor))
static void __init__(int argc, const char **argv)
{
     printf("# app: %s\n",argv[0]);
     printf("# lib: %s\n",getenv("DYLD_INSERT_LIBRARIES"));
     syslog(LOG_INFO, "Dylib injection successful in %s\n", argv[0]);
     s_bActive = true;
}

int my_getnameinfo(const struct sockaddr *addr, socklen_t addrlen,
                char *host, socklen_t hostlen,
                char *serv, socklen_t servlen, int flags)
{
    if (!s_bActive)
    {
        return getnameinfo(addr, addrlen, host, hostlen, serv, servlen, flags);
    }
    return getnameinfo(addr, addrlen, host, hostlen, serv, servlen, flags);
}

hostent * my_gethostbyname(const char * name)
{
    if (!s_bActive)
    {
        return gethostbyname(name);
    }
    printf("gethostbyname(%s)...\n",name);
    return gethostbyname(name);
}

int my_getaddrinfo(const char *node,
                       const char * service,
                       const struct addrinfo * hints,
                       struct addrinfo ** res)
{
    if (!s_bActive)
    {
        return getaddrinfo(node,service,hints,res);
    }
    printf("getaddrinfo(%s,%s)...\n",node,service);
    return getaddrinfo(node,service,hints,res);
}

extern "C" int my_connect(int fd, const struct sockaddr * addr, socklen_t len)
{
    if (!s_bActive)
    {
        return connect(fd,addr,len);
    }
    printf("connect...\n");
    if (addr->sa_family == AF_INET) {
        char szAddr[32]="";
        struct sockaddr_in * addr_in = (struct sockaddr_in *)addr;
        inet_ntop(AF_INET, &(addr_in->sin_addr), szAddr, 31);
        printf("%s:%i\n",szAddr,addr_in->sin_port);
    }

    return connect(fd,addr,len);
}

#define DYLD_INTERPOSE(_replacement,_replacee) \
   __attribute__((used)) static struct{ const void* replacement; const void* replacee; } _interpose_##_replacee \
            __attribute__ ((section ("__DATA,__interpose"))) = { (const void*)(unsigned long)&_replacement, (const void*)(unsigned long)&_replacee };

DYLD_INTERPOSE(my_connect, connect)
DYLD_INTERPOSE(my_gethostbyname, gethostbyname)
DYLD_INTERPOSE(my_getaddrinfo, getaddrinfo)


