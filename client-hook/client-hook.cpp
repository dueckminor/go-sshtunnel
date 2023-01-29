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
#include <string.h>
#include <errno.h>
#include <fcntl.h>

static bool s_bActive = false;
static int s_port = 0;

__attribute__((constructor))
static void __init__(int argc, const char **argv)
{
    //printf("# app: %s\n",argv[0]);
    const char * proxy = getenv("SSHTUNNEL_PROXY");
    //printf("# prx: %s\n",proxy);

    if (0==strncmp(proxy,"http://localhost:",17))
    {
        s_port = atoi(proxy+17);
        //printf("# port: %d\n",s_port);
    }

     syslog(LOG_INFO, "Dylib injection successful in %s\n", argv[0]);
     s_bActive = true;
}

class HttpClient
{
public:
    HttpClient(int fd=-1)
    {
        m_fd = fd;
        m_dont_close = (fd>=0);

        memset(m_buffer,0,sizeof(m_buffer));
        memset(m_keys,0,sizeof(m_keys));
        memset(m_values,0,sizeof(m_values));
        m_nvalues=0;
    }

    ~HttpClient()
    {
        if (m_dont_close)
        {
            return;
        }
        if (m_fd >= 0)
        {
            close(m_fd);
        }
    }
    bool Connect()
    {
        if (m_fd < 0)
        {
            m_fd = socket(AF_INET, SOCK_STREAM, 0);
            if (m_fd < 0)
            {
                //printf("\n socket failed \n");
                return false;
            }
        }
        if (!m_connected)
        {
            struct sockaddr_in serv_addr;
            serv_addr.sin_family = AF_INET;
            serv_addr.sin_port = htons(s_port);
            if (inet_pton(AF_INET, "127.0.0.1", &serv_addr.sin_addr) <= 0)
            {
                //printf("\n inet_pton failed \n");
                return false;
            }
            int rc = connect(m_fd, (struct sockaddr*)&serv_addr, sizeof(serv_addr));
            if (rc < 0)
            {
                return false;
            }
            m_connected = true;
        }
        return true;
    }
    int Request(const char * method, const char * path, const char * hostname)
    {
        if (!Connect())
        {
            return -1;
        }
        memset(m_buffer,0,sizeof(m_buffer));
        snprintf(m_buffer,sizeof(m_buffer),"%s %s HTTP/1.1\r\nHost: %s\r\n\r\n",method,path,hostname);

        m_nvalues=0;
        memset(m_keys,0,sizeof(m_keys));
        memset(m_values,0,sizeof(m_values));

        int n=send(m_fd, m_buffer, strlen(m_buffer), 0);
        memset(m_buffer,0,sizeof(m_buffer));

        for (char * p = m_buffer;p<m_buffer+sizeof(m_buffer)-1;)
        {
            if (!read(m_fd,p,1))
            {
                break;
            }
            switch(*p)
            {
            case '\n':
                *p = '\0';
                m_nvalues++;
                m_keys[m_nvalues]=p+1;
                break;
            case ':':
                if (!m_values[m_nvalues])
                {
                    *p = '\0';
                    m_values[m_nvalues] = p+2;
                }
                break;
            case '\r':
                p--; // remove from m_buffer
                break;
            }
            if ((m_nvalues>1) && !m_keys[m_nvalues-1][0])
            {
                break;
            }
            p++;
        }

        //printf("RESPONSE: %s\n",m_buffer);
        // for (int i=1;i<m_nvalues;i++)
        // {
        //     printf("'%s'='%s'\n",m_keys[i],m_values[i]);
        // }
        //printf(">>>>\n");
        return 0;
    }
    const char * GetHeader(const char * key)
    {
        for (int i=1;i<m_nvalues;i++)
        {
            if (0==strcmp(m_keys[i],key))
            {
                if (!m_values[i][0])
                {
                    return NULL;
                }
                return m_values[i];
            }
        }
        return NULL;
    }

protected:
    bool m_dont_close;
    bool m_connected;
    int m_fd;
    char m_buffer[1024];
    const char* m_keys[32];
    const char* m_values[32];
    int m_nvalues;
};

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

// hostent * my_gethostbyname(const char * name)
// {
//     if (!s_bActive)
//     {
//         return gethostbyname(name);
//     }
//     //printf("gethostbyname(%s)...\n",name);
//     return gethostbyname(name);
// }

bool getaddr(const char *hostname, const char * service, struct addrinfo ** res)
{
    HttpClient client;
    client.Request("RESOLVE","*",hostname);

    const char * ip = client.GetHeader("Host");
    if (!ip)
    {
        return false;
    }

    //printf("'%s' -> '%s:%s'\n",hostname,ip,service);

    return (0 == getaddrinfo(ip,service,NULL,res));
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
    //printf("getaddrinfo(%s,%s)...\n",node,service);

    if (getaddr(node,service,res))
    {
        return 0;
    }

    return getaddrinfo(node,service,hints,res);
}

int socket_get_type(int fd)
{
    int sock_type = -1;
    socklen_t sock_type_len = sizeof(sock_type);
    getsockopt(fd, SOL_SOCKET, SO_TYPE,
        (void *) &sock_type, &sock_type_len);
    return sock_type;
}

class SocketMakeBlocking
{
public:
    SocketMakeBlocking(int fd)
    {
        m_fd = fd;
        m_flags = fcntl(m_fd, F_GETFL);
        if (m_flags & O_NONBLOCK)
        {
            fcntl(m_fd, F_SETFL, m_flags & ~O_NONBLOCK);
        }
    }
    ~SocketMakeBlocking()
    {
        if (m_flags & O_NONBLOCK)
        {
            fcntl(m_fd, F_SETFL, m_flags);
        }
    }
protected:
    int m_fd;
    int m_flags;
};

extern "C" int my_connect(int fd, const struct sockaddr * addr, socklen_t len)
{
    if (!s_bActive)
    {
        return connect(fd,addr,len);
    }
    //printf("connect(%d,...)...\n",fd);

    if ((addr->sa_family != AF_INET) || (socket_get_type(fd) != SOCK_STREAM))
    {
        return connect(fd,addr,len);
    }

    SocketMakeBlocking make_blocking(fd);

    char szHostIP[32]="";
    struct sockaddr_in * addr_in = (struct sockaddr_in *)addr;
    inet_ntop(AF_INET, &(addr_in->sin_addr), szHostIP, 31);
    char szHostPort[64]="";
    snprintf(szHostPort,63,"%s:%i",szHostIP,ntohs(addr_in->sin_port));

    HttpClient client(fd);
    return client.Request("CONNECT",szHostPort,szHostPort);
}

#define DYLD_INTERPOSE(_replacement,_replacee) \
   __attribute__((used)) static struct{ const void* replacement; const void* replacee; } _interpose_##_replacee \
            __attribute__ ((section ("__DATA,__interpose"))) = { (const void*)(unsigned long)&_replacement, (const void*)(unsigned long)&_replacee };

DYLD_INTERPOSE(my_connect, connect)
// DYLD_INTERPOSE(my_gethostbyname, gethostbyname)
DYLD_INTERPOSE(my_getaddrinfo, getaddrinfo)


