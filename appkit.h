#ifndef KANJIBAR_APPKIT_H
#define KANJIBAR_APPKIT_H

#ifdef __cplusplus
extern "C" {
#endif

/* Called from Go. All strings are UTF-8 and may be freed by the caller as soon
   as these functions return (the Objective-C side copies them immediately). */

void KBStart(void);          /* create status item, panel, menu, key monitor   */
void KBRun(void);            /* run the AppKit event loop                      */
void KBShowCard(const char *cardJSON);
void KBSetTitle(const char *title);
void KBSetMenuState(int autoOn, int romajiOn, int furiganaOn, int level);
double KBCardNaturalHeight(const char *cardJSON);
void KBSetFixedCardHeight(double height);
int KBIsPanelVisible(void);
void KBSnapshot(const char *cardJSON, const char *path, int dark);
void KBShowNow(void);        /* dev: open the panel shortly after launch      */
void KBHideNow(void);        /* dev: close the panel                         */
void KBQuit(void);           /* dev: terminate the app                       */

/* Implemented in Go; called from Objective-C on the main thread. */

void GoNext(void);
void GoPrev(void);
void GoToggleMode(void);
void GoToggleAuto(void);
void GoToggleRomaji(void);
void GoToggleFurigana(void);
void GoCycleLevel(void);
void GoSetLevel(int level);

#ifdef __cplusplus
}
#endif

#endif /* KANJIBAR_APPKIT_H */
