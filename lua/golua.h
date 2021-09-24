#include "lua/lua.h"
#include "lua/lauxlib.h"
#include "lua/lualib.h"
#define LJ_HASFFI 1
#include "lua/luajit-ffi-ctypeid.h"
#include <stdint.h>
#include <stdlib.h> // _atoi64 on windows, atoll on posix.
#include <stdio.h>

#ifndef golua_h
#define golua_h

typedef struct { int t; void *v; } GoValue;

#define GOLUA_DEFAULT_MSGHANDLER "golua_default_msghandler"

/* function to setup metatables, etc */
void clua_initstate(lua_State* L);
void clua_hide_pcall(lua_State *L);

lua_State* clua_newstate(void* goallocf);
int clua_setgostate(lua_State* L, size_t gostateindex);
size_t clua_getgostate(lua_State* L);
int clua_dedup_coro(lua_State* coro);

int clua_togofunction(lua_State* L, int index);
int clua_togostruct(lua_State *L, int index);

void clua_pushcallback(lua_State* L);
void clua_pushgofunction(lua_State* L, unsigned int fid);
void clua_pushgostruct(lua_State *L, unsigned int fid);

int dump_chunk (lua_State *L);
int load_chunk(lua_State *L, char *b, int size, const char* chunk_name);
GoValue clua_atpanic(lua_State* L, unsigned int panicf_id);
int clua_callluacfunc(lua_State* L, lua_CFunction f);

void clua_setallocf(lua_State* L, void* goallocf);
void clua_sethook(lua_State* L, int n);

void clua_openbase(lua_State* L);
void clua_openio(lua_State* L);
void clua_openmath(lua_State* L);
void clua_openpackage(lua_State* L);
void clua_openstring(lua_State* L);
void clua_opentable(lua_State* L);
void clua_openos(lua_State* L);

uint32_t clua_luajit_ctypeid(lua_State *L, int idx);
void clua_luajit_push_cdata_int64(lua_State *L, int64_t n);
void clua_luajit_push_cdata_uint64(lua_State *L, uint64_t u);

int clua_isgofunction(lua_State *L, int n);
int clua_isgostruct(lua_State *L, int n);

void bundle_add_loaders(lua_State* L);
int bundle_main(lua_State *L, int argc, char** argv);

#endif
