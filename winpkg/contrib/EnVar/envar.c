/*
 * EnVar plugin for NSIS
 *
 * 2014-2016, 2018 MouseHelmet Software.
 *
 * Created By Jason Ross aka JasonFriday13 on the forums
 *
 * Checks, adds and removes paths to environment variables.
 *
 * envar.c
 */

/* Include relevent files. */
#include <windows.h>
#include <shlwapi.h>
#include "nsis\pluginapi.h" /* This means NSIS 2.42 or higher is required. */

/* Registry defines. */
#define HKCU HKEY_CURRENT_USER
#define HKCU_STR _T("Environment")
#define HKLM HKEY_LOCAL_MACHINE
#define HKLM_STR _T("System\\CurrentControlSet\\Control\\Session Manager\\Environment")

/* I would have used ints, but removing pushint() also
   removes a dependency on wsprintf and user32.dll. */
#define ERR_SUCCESS _T("0")
#define ERR_NOREAD _T("1")
#define ERR_NOVARIABLE _T("2")
#define ERR_NOVALUE _T("3")
#define ERR_NOWRITE _T("4")

/* The amount of extra room to allocate to prevent overflows. */
#define APPEND_SIZE 4

/* Unicode and odd value finder. */
#define IS_UNICODE_AND_ODD(x) ((sizeof(TCHAR) > 1 && (x) & 0x1) ? 1 : 0)

/* Global declarations. */
BOOL bRegKeyHKLM = FALSE;
HINSTANCE hInstance;
PTCHAR gptVarName, gptPathString, gptBuffer;

/* Allocates a string. */
PTCHAR StrAlloc(SIZE_T strlen)
{
  return (PTCHAR)GlobalAlloc(GPTR, strlen*sizeof(TCHAR));
}

/* Frees a string. */
void StrFree(PTCHAR hVar)
{
  GlobalFree(hVar);
  hVar = NULL;
}

/* Returns the string size. */
int StrSize(PTCHAR hVar)
{
  return (int)((GlobalSize(hVar)-IS_UNICODE_AND_ODD(GlobalSize(hVar)))/sizeof(TCHAR));
}

/* Returns the string length. */
int StrLen(PTCHAR hVar)
{
  return lstrlen(hVar);
}

/* Reallocs a buffer. It's more efficient to free and alloc than realloc. */ 
BOOL StrReAlloc(PTCHAR hVar, int strlen)
{
  int i;
  PTCHAR temp;

  if (GlobalSize(hVar) >= strlen*sizeof(TCHAR)) return 1;
  temp = StrAlloc(strlen);
  if (!temp) return 0;
  for (i = 0; i < StrSize(hVar); i++)
    temp[i] = hVar[i];
  StrFree(hVar);
  hVar = temp;

  return 1;
}

/* Initializes the default string size for variables. */
void AllocStrs(void)
{
  if (!gptVarName) gptVarName = StrAlloc(g_stringsize);
  if (!gptPathString) gptPathString = StrAlloc(g_stringsize);
  if (!gptBuffer) gptBuffer = StrAlloc(g_stringsize);
}

/* Frees allocated buffers. */
void CleanUp(void)
{
  if (gptVarName) StrFree(gptVarName);
  if (gptPathString) StrFree(gptPathString);
  if (gptBuffer) StrFree(gptBuffer);

  SendMessageTimeout(HWND_BROADCAST, WM_SETTINGCHANGE, (WPARAM)NULL, (LPARAM)HKCU_STR, 0, 100, 0);
}

/*  Our callback function so that our dll stays loaded. */
UINT_PTR __cdecl NSISPluginCallback(enum NSPIM Event) 
{
  if (Event == NSPIM_UNLOAD) CleanUp();

  return 0;
}

/* This function initializes the plugin variables. */
void Initialize(extra_parameters* xp)
{
  xp->RegisterPluginCallback(hInstance, NSISPluginCallback);
  AllocStrs();
}

/* Appends a semi-colon to a string. Auto-expands the buffer
   if there isn't enough room. */
BOOL AppendSemiColon(PTCHAR bufStr)
{
  int len;
  
  if (!bufStr) return FALSE;
  len = StrLen(bufStr);
  if (!len) return TRUE;
  if (bufStr[len-1] != ';' && bufStr[0] != ';')
  {
    if (!StrReAlloc(bufStr, len+APPEND_SIZE)) return FALSE;
    bufStr[len] = ';';
    bufStr[len+1] = 0;
  }
  return TRUE;
}

/* Removes the trailing semi-colon if it exists. */
void RemoveSemiColon(PTCHAR bufStr)
{
  if (!bufStr) return;
  if (StrLen(bufStr) < 1) return;
  if (bufStr[StrLen(bufStr)-1] == ';') bufStr[StrLen(bufStr)-1] = 0;
}

/* Sets the current root key. */
void SetRegKey(BOOL bKeyHKLM)
{
  bRegKeyHKLM = bKeyHKLM;
}

/* Gets the current root key. */
BOOL GetRegKey(void)
{
  return bRegKeyHKLM;
}

/* Registry helper functions. */
ULONG CreateRegKey(void)
{
  DWORD dwRet, dwDisType = 0;
  HKEY hKey;

  if (bRegKeyHKLM)
    dwRet = RegCreateKeyEx(HKLM, HKLM_STR, 0, 0, 0, KEY_WRITE, 0, &hKey, &dwDisType);
  else
    dwRet = RegCreateKeyEx(HKCU, HKCU_STR, 0, 0, 0, KEY_WRITE, 0, &hKey, &dwDisType);

  RegCloseKey(hKey);

  return dwRet;
}

/* Custom ReadRegVar function with ERROR_MORE_DATA handling. */
ULONG ReadRegVar(PTCHAR ptName, PDWORD pdwType, PTCHAR ptDest, PDWORD pdwStrLen)
{
  DWORD dwRet, dwSize = *pdwStrLen*sizeof(TCHAR), dwType = *pdwType;
  HKEY hKey;

  ptDest[0] = 0;
  if (bRegKeyHKLM)
    dwRet = RegOpenKeyEx(HKLM, HKLM_STR, 0, KEY_READ, &hKey);
  else
    dwRet = RegOpenKeyEx(HKCU, HKCU_STR, 0, KEY_READ, &hKey);

  if (dwRet == ERROR_SUCCESS)
  {
    dwRet = RegQueryValueEx(hKey, ptName, 0, &dwType, (LPBYTE)ptDest, &dwSize);
    while (dwRet == ERROR_MORE_DATA)
    {
      DWORD dwSizeTemp = dwSize + APPEND_SIZE + IS_UNICODE_AND_ODD(dwSize);

      if (!StrReAlloc(ptDest, dwSizeTemp/sizeof(TCHAR))) return 4; /* ERR_NOWRITE */
      dwRet = RegQueryValueEx(hKey, ptName, 0, &dwType, (LPBYTE)ptDest, &dwSizeTemp);
      if (dwRet == ERROR_SUCCESS) dwSize = dwSizeTemp;
    }
    RegCloseKey(hKey);
    if (dwRet == ERROR_SUCCESS && (dwType == REG_SZ || dwType == REG_EXPAND_SZ))
    {
      ptDest[((dwSize+IS_UNICODE_AND_ODD(dwSize))/sizeof(TCHAR))-1] = 0;
      *pdwType = dwType;
      *pdwStrLen = (dwSize-IS_UNICODE_AND_ODD(dwSize))/sizeof(TCHAR);

      return 0;
    }
    else
    {
      ptDest[0] = 0;
      /* dwRet can still be ERROR_SUCCESS here, so return an absolute value. */
      return 1; /* ERR_NOREAD */
    }
  }
  else
    return 1; /* ERR_NOREAD */
}

/* Custom WriteRegVar function, writes a value to the environment. */
ULONG WriteRegVar(PTCHAR ptName, DWORD dwKeyType, PTCHAR ptData, DWORD dwStrLen)
{
  DWORD dwRet;
  HKEY hKey;

  if (bRegKeyHKLM)
    dwRet = RegOpenKeyEx(HKLM, HKLM_STR, 0, KEY_WRITE, &hKey);
  else
    dwRet = RegOpenKeyEx(HKCU, HKCU_STR, 0, KEY_WRITE, &hKey);

  if (dwRet != ERROR_SUCCESS) return dwRet;
  dwRet = RegSetValueEx(hKey, ptName, 0, dwKeyType, (LPBYTE)ptData, dwStrLen*sizeof(TCHAR));
  RegCloseKey(hKey);

  return dwRet;
}

/* Checks for write access and various conditions about a variable and it's type. */
LPCTSTR CheckVar(void)
{
  DWORD dwStrSize, dwKeyType;
  HKEY hKeyHandle;

  SecureZeroMemory(gptBuffer, GlobalSize(gptBuffer));

  popstring(gptVarName);
  popstring(gptPathString);
  
  if (!StrLen(gptVarName)) return ERR_NOVARIABLE;
  if (lstrcmpi(gptVarName, _T("NULL")) == 0)
  {
    DWORD dwRet;

    if (bRegKeyHKLM)
      dwRet = RegOpenKeyEx(HKLM, HKLM_STR, 0, KEY_WRITE, &hKeyHandle);
    else
      dwRet = RegOpenKeyEx(HKCU, HKCU_STR, 0, KEY_WRITE, &hKeyHandle);

    if (dwRet == ERROR_SUCCESS)
    {
      RegCloseKey(hKeyHandle);
      return ERR_SUCCESS;
    }
    else
      return ERR_NOWRITE;
  }
  dwStrSize = StrSize(gptBuffer);
  if (ReadRegVar(gptVarName, &dwKeyType, gptBuffer, &dwStrSize) != ERROR_SUCCESS)
    return ERR_NOVARIABLE;

  if (!StrLen(gptPathString)) return ERR_NOVARIABLE;
  if (lstrcmpi(gptPathString, _T("NULL")) != 0)
  {
    if (!AppendSemiColon(gptPathString)) return ERR_NOWRITE;
    if (!AppendSemiColon(gptBuffer)) return ERR_NOWRITE;
    if (StrStrI(gptBuffer, gptPathString) == NULL)
      return ERR_NOVALUE;
    else
      return ERR_SUCCESS;
  }
  else
    if (dwKeyType != REG_SZ && dwKeyType != REG_EXPAND_SZ)
      return ERR_NOVALUE;
    else
      return ERR_SUCCESS;
}

/* Adds a value to a variable if it's the right type. */
LPCTSTR AddVarValue(DWORD dwKey)
{
  DWORD dwStrSize;

  SecureZeroMemory(gptBuffer, GlobalSize(gptBuffer));

  popstring(gptVarName);
  popstring(gptPathString);

  if (!StrLen(gptPathString)) return ERR_NOVALUE;

  if (CreateRegKey() != ERROR_SUCCESS)
    return ERR_NOWRITE;
  else
  {
    DWORD dwKeyType;

    dwStrSize = StrSize(gptBuffer);
    ReadRegVar(gptVarName, &dwKeyType, gptBuffer, &dwStrSize);

    if (dwKeyType == REG_EXPAND_SZ) dwKey = dwKeyType;
    if (!AppendSemiColon(gptPathString)) return ERR_NOWRITE;
    if (!AppendSemiColon(gptBuffer)) return ERR_NOWRITE;

    if (StrStrI(gptBuffer, gptPathString) == NULL)
    {
      int i, len = StrLen(gptBuffer);

      /* Add one for separator and one for terminating NULL character. */
      if (!StrReAlloc(gptBuffer, len+StrLen(gptPathString)+APPEND_SIZE))
        return ERR_NOWRITE;

      for (i = 0; i <= StrLen(gptPathString); i++)
        gptBuffer[len+i] = gptPathString[i];

      RemoveSemiColon(gptBuffer);
      if (WriteRegVar(gptVarName, dwKey, gptBuffer, StrLen(gptBuffer)+1) != ERROR_SUCCESS)
        return ERR_NOWRITE;
      else
        return ERR_SUCCESS;
    }
    else
      return ERR_SUCCESS;
  }
}

/* Deletes a value from a variable if it's the right type. */
LPCTSTR DeleteVarValue(void)
{
  DWORD dwStrSize, dwKeyType;

  SecureZeroMemory(gptBuffer, GlobalSize(gptBuffer));

  popstring(gptVarName);
  popstring(gptPathString);

  dwStrSize = StrSize(gptBuffer);
  if (ReadRegVar(gptVarName, &dwKeyType, gptBuffer, &dwStrSize) != ERROR_SUCCESS)
    return ERR_NOVARIABLE;

  if (!AppendSemiColon(gptPathString)) return ERR_NOWRITE;
  if (!AppendSemiColon(gptBuffer)) return ERR_NOWRITE;

  if (StrStrI(gptBuffer, gptPathString) == NULL)
    return ERR_NOVALUE;
  else
  {
    do
    {
      int i, len;
      const PTCHAR str = StrStrI(gptBuffer, gptPathString);

      if (str[StrLen(gptPathString)] == ';')
      {
        len = StrLen(str);
        for (i = StrLen(gptPathString); i < len; i++)
          str[i] = str[i+1];
      }
      len = StrLen(gptBuffer) - StrLen(str);
      for (i = StrLen(gptPathString); i <= StrLen(str); i++, len++)
        gptBuffer[len] = str[i];

    } while (StrStrI(gptBuffer, gptPathString) != NULL);
    
    RemoveSemiColon(gptBuffer);
    if (WriteRegVar(gptVarName, dwKeyType, gptBuffer, StrLen(gptBuffer)+1) != ERROR_SUCCESS)
      return ERR_NOWRITE;
    else
      return ERR_SUCCESS;
  }
}

/* Deletes a variable from the environment. */
LPCTSTR DeleteVar(void)
{
  DWORD dwRet, res;
  HKEY hKey;

  popstring(gptVarName);

  if (!lstrcmpi(gptVarName, _T("path")))
    return ERR_NOWRITE;
  
  if (bRegKeyHKLM)
    dwRet = RegOpenKeyEx(HKLM, HKLM_STR, 0, KEY_WRITE, &hKey);
  else
    dwRet = RegOpenKeyEx(HKCU, HKCU_STR, 0, KEY_WRITE, &hKey);

  if (dwRet == ERROR_SUCCESS)
  {
    res = RegDeleteValue(hKey, gptVarName);
    RegCloseKey(hKey);
    if (res == ERROR_SUCCESS)
      return ERR_SUCCESS;
  }

  return ERR_NOWRITE;
}

/* Updates the installer environment from the registry. */
LPCTSTR UpdateVar(void)
{
  PTCHAR ptRegRoot = gptVarName, ptVarName = gptPathString;
  DWORD dwRet, dwStrSize, dwKeyType;
  BOOL bOldKey = GetRegKey();
  int i;

  popstring(ptRegRoot);
  popstring(ptVarName);

  if (!lstrcmpi(ptRegRoot, _T("HKCU")) || !lstrcmpi(ptRegRoot, _T("HKLM")))
  {
    if (!lstrcmpi(ptRegRoot, _T("HKLM")))
      SetRegKey(TRUE);
    else
      SetRegKey(FALSE);

    dwRet = ReadRegVar(ptVarName, &dwKeyType, gptBuffer, &dwStrSize);
    SetRegKey(bOldKey);
    if (dwRet != ERROR_SUCCESS)
      return ERR_NOVARIABLE;
  }
  else
  {
    int len;

    SetRegKey(FALSE);
    dwRet = ReadRegVar(ptVarName, &dwKeyType, gptBuffer, &dwStrSize);
    if (dwRet != ERROR_SUCCESS)
      *ptRegRoot = 0;
    else
    {
      if (!StrReAlloc(ptRegRoot, StrLen(gptBuffer)+APPEND_SIZE))
      {
        SetRegKey(bOldKey);
        return ERR_NOWRITE;
      }
      /* Update global pointer if ptRegRoot was changed. */
      gptVarName = (gptVarName != ptRegRoot) ? ptRegRoot : gptVarName;
      for (i = 0; i <= StrLen(gptBuffer); i++)
        ptRegRoot[i] = gptBuffer[i];
      AppendSemiColon(ptRegRoot);
    }
    SetRegKey(TRUE);
    dwRet = ReadRegVar(ptVarName, &dwKeyType, gptBuffer, &dwStrSize);
    SetRegKey(bOldKey);
    if (dwRet != ERROR_SUCCESS)
      if (!(*ptRegRoot))
        return ERR_NOVARIABLE;
      else
        *gptBuffer = 0;

    AppendSemiColon(gptBuffer);
    len = StrLen(gptBuffer);

    /* Add one for separator and one for terminating NULL character. */
    if (!StrReAlloc(gptBuffer, len+StrLen(ptRegRoot)+APPEND_SIZE))
      return ERR_NOWRITE;

    for (i = 0; i <= StrLen(ptRegRoot); i++)
      gptBuffer[len+i] = ptRegRoot[i];

    RemoveSemiColon(gptBuffer);
  }
  if (SetEnvironmentVariable(ptVarName, gptBuffer))
    return ERR_SUCCESS;
  else
    return ERR_NOWRITE;
}

/* This routine sets the environment root, HKCU. */
__declspec(dllexport) void SetHKCU(HWND hwndParent, int string_size, TCHAR *variables, stack_t **stacktop, extra_parameters* xp)
{
  /* Initialize the stack so we can access it from our DLL using 
  popstring and pushstring. */
  EXDLL_INIT();
  Initialize(xp);

  SetRegKey(FALSE);
}

/* This routine sets the environment root, HKLM. */
__declspec(dllexport) void SetHKLM(HWND hwndParent, int string_size, TCHAR *variables, stack_t **stacktop, extra_parameters* xp)
{
  /* Initialize the stack so we can access it from our DLL using 
  popstring and pushstring. */
  EXDLL_INIT();
  Initialize(xp);

  SetRegKey(TRUE);
}

/* This routine checks for a path in an environment variable. */
__declspec(dllexport) void Check(HWND hwndParent, int string_size, TCHAR *variables, stack_t **stacktop, extra_parameters* xp)
{
  /* Initialize the stack so we can access it from our DLL using 
  popstring and pushstring. */
  EXDLL_INIT();
  Initialize(xp);

  pushstring(CheckVar());
}

/* This routine adds a REG_SZ value in a environment variable (checks for existing paths first). */
__declspec(dllexport) void AddValue(HWND hwndParent, int string_size, TCHAR *variables, stack_t **stacktop, extra_parameters* xp)
{
  /* Initialize the stack so we can access it from our DLL using 
  popstring and pushstring. */
  EXDLL_INIT();
  Initialize(xp);

  pushstring(AddVarValue(REG_SZ));
}

/* This routine adds a REG_EXPAND_SZ value in a environment variable (checks for existing paths first). */
__declspec(dllexport) void AddValueEx(HWND hwndParent, int string_size, TCHAR *variables, stack_t **stacktop, extra_parameters* xp)
{
  /* Initialize the stack so we can access it from our DLL using 
  popstring and pushstring. */
  EXDLL_INIT();
  Initialize(xp);

  pushstring(AddVarValue(REG_EXPAND_SZ));
}

/* This routine deletes a value in an environment variable if it exists. */
__declspec(dllexport) void DeleteValue(HWND hwndParent, int string_size, TCHAR *variables, stack_t **stacktop, extra_parameters* xp)
{
  /* Initialize the stack so we can access it from our DLL using 
  popstring and pushstring. */
  EXDLL_INIT();
  Initialize(xp);

  pushstring(DeleteVarValue());
}

/* This routine deletes an environment variable if it exists. */
__declspec(dllexport) void Delete(HWND hwndParent, int string_size, TCHAR *variables, stack_t **stacktop, extra_parameters* xp)
{
  /* Initialize the stack so we can access it from our DLL using 
  popstring and pushstring. */
  EXDLL_INIT();
  Initialize(xp);

  pushstring(DeleteVar());
}

/* This routine reads the registry and updates the process environment. */
__declspec(dllexport) void Update(HWND hwndParent, int string_size, TCHAR *variables, stack_t **stacktop, extra_parameters* xp)
{
  /* Initialize the stack so we can access it from our DLL using 
  popstring and pushstring. */
  EXDLL_INIT();
  Initialize(xp);

  pushstring(UpdateVar());
}

/* Our DLL entry point, this is called when we first load up our DLL. */
BOOL WINAPI _DllMainCRTStartup(HINSTANCE hInst, DWORD ul_reason_for_call, LPVOID lpReserved)
{
  hInstance = hInst;

  if (ul_reason_for_call == DLL_PROCESS_DETACH)
    CleanUp();

  return TRUE;
}