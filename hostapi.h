#ifndef MYSTREAM_BOT_HOST_API_H
#define MYSTREAM_BOT_HOST_API_H

#include <stdint.h>

#define MYSTREAM_BOT_API_VERSION 0x00010000

typedef struct bot_host_api_s {
	uint32_t api_version;
	void     (*log)(int level, char *msg);
	char *   (*emit_event)(char *type, char *data, char *target);
	void     (*free)(void *ptr);
} bot_host_api_t;

#endif
