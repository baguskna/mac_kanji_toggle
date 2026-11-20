#import <Cocoa/Cocoa.h>
#import <QuartzCore/QuartzCore.h>
#import "appkit.h"

#pragma mark - Layout constants

static const CGFloat kCardWidth    = 380.0; // visible card width
static const CGFloat kShadowMargin = 10.0;  // transparent margin that holds the window shadow
static const CGFloat kPadding      = 20.0;  // card inner padding
static const CGFloat kPillHeight   = 20.0;


static const CGFloat kContentWidth = kCardWidth - 2 * kPadding;
static const CGFloat kViewWidth    = kCardWidth + 2 * kShadowMargin;

#pragma mark - Small helpers

static NSString *Str(NSDictionary *d, NSString *key) {
    id v = d[key];
    if ([v isKindOfClass:[NSString class]]) return v;
    if ([v isKindOfClass:[NSNumber class]]) return [v stringValue];
    return @"";
}

static NSFont *SysFont(CGFloat size, NSFontWeight weight) {
    return [NSFont systemFontOfSize:size weight:weight];
}

static NSFont *MonoFont(CGFloat size) {
    return [NSFont monospacedSystemFontOfSize:size weight:NSFontWeightRegular];
}

static NSMutableParagraphStyle *Para(NSTextAlignment align) {
    NSMutableParagraphStyle *p = [[[NSMutableParagraphStyle alloc] init] autorelease];
    p.alignment = align;
    p.lineBreakMode = NSLineBreakByWordWrapping;
    return p;
}

static NSMutableAttributedString *Text(NSString *s, NSFont *font, NSColor *color,
                                       NSTextAlignment align, CGFloat kern) {
    NSMutableParagraphStyle *p = Para(align);
    NSMutableDictionary *attrs = [NSMutableDictionary dictionary];
    attrs[NSFontAttributeName] = font;
    attrs[NSForegroundColorAttributeName] = color;
    attrs[NSParagraphStyleAttributeName] = p;
    if (kern != 0.0) attrs[NSKernAttributeName] = @(kern);
    return [[[NSMutableAttributedString alloc] initWithString:(s ?: @"") attributes:attrs] autorelease];
}

static CGFloat TextHeight(NSAttributedString *a, CGFloat width) {
    NSRect r = [a boundingRectWithSize:NSMakeSize(width, CGFLOAT_MAX)
                               options:(NSStringDrawingUsesLineFragmentOrigin | NSStringDrawingUsesFontLeading)];
    return ceil(r.size.height);
}

static CGFloat TextWidth(NSAttributedString *a) {
    return ceil([a size].width);
}

static void Draw(NSAttributedString *a, NSRect rect) {
    [a drawWithRect:rect
            options:(NSStringDrawingUsesLineFragmentOrigin | NSStringDrawingUsesFontLeading)];
}

static void DrawPill(NSRect rect, NSColor *fill, NSString *text, NSFont *font,
                     NSColor *fg, CGFloat kern) {
    NSBezierPath *p = [NSBezierPath bezierPathWithRoundedRect:rect
                                                      xRadius:rect.size.height / 2.0
                                                      yRadius:rect.size.height / 2.0];
    [fill setFill];
    [p fill];
    NSAttributedString *a = Text(text, font, fg, NSTextAlignmentCenter, kern);
    CGFloat h = TextHeight(a, rect.size.width);
    Draw(a, NSMakeRect(rect.origin.x, NSMidY(rect) - h / 2.0, rect.size.width, h));
}

static void DrawButton(NSRect rect, NSString *title, BOOL primary) {
    NSBezierPath *p = [NSBezierPath bezierPathWithRoundedRect:rect xRadius:8 yRadius:8];
    NSColor *fill = primary ? [[NSColor controlAccentColor] colorWithAlphaComponent:0.16]
                            : [[NSColor labelColor] colorWithAlphaComponent:0.06];
    [fill setFill];
    [p fill];
    NSColor *stroke = primary ? [[NSColor controlAccentColor] colorWithAlphaComponent:0.35]
                              : [[NSColor labelColor] colorWithAlphaComponent:0.12];
    [stroke setStroke];
    p.lineWidth = 1.0;
    [p stroke];

    NSColor *fg = primary ? [NSColor controlAccentColor] : [NSColor labelColor];
    NSAttributedString *a = Text(title, SysFont(12.0, NSFontWeightSemibold), fg,
                                 NSTextAlignmentCenter, 0);
    CGFloat h = TextHeight(a, rect.size.width);
    Draw(a, NSMakeRect(rect.origin.x, NSMidY(rect) - h / 2.0, rect.size.width, h));
}

static void DrawSeparator(NSRect rect) {
    [[[NSColor separatorColor] colorWithAlphaComponent:0.8] setFill];
    NSRectFill(rect);
}

static void CopyCardToPasteboard(NSDictionary *card) {
    NSString *head    = Str(card, @"headword");
    NSString *reading = Str(card, @"reading1");
    NSString *romaji  = Str(card, @"reading1Romaji");
    NSString *meaning = Str(card, @"meaning");
    if (head.length == 0) return;

    NSMutableString *s = [NSMutableString stringWithString:head];
    if (reading.length > 0) {
        [s appendFormat:@"（%@", reading];
        if (romaji.length > 0) [s appendFormat:@" / %@", romaji];
        [s appendString:@"）"];
    }
    if (meaning.length > 0) [s appendFormat:@" — %@", meaning];

    NSPasteboard *pb = [NSPasteboard generalPasteboard];
    [pb clearContents];
    [pb setString:s forType:NSPasteboardTypeString];
}

#pragma mark - Forward declarations

static void ShowPopover(void);
static void HidePopover(BOOL restoreFocus);
static void HandleStatusItemClick(void);
static NSEvent *HandleKey(NSEvent *event);

#pragma mark - Card view

@interface KBCardView : NSView {
    NSDictionary *_card;
    BOOL _isKanji;
    CGFloat _height;

    NSRect _cardRect;
    NSRect _modePill;
    NSRect _badgePill;
    NSRect _headwordRect;
    NSRect _metaRect;
    NSRect _r1LabelRect, _r1ValueRect, _r1RomajiRect;
    NSRect _r2LabelRect, _r2ValueRect, _r2RomajiRect;
    NSRect _meaningRect;
    NSRect _divider1Rect;
    NSRect _sectionRect;
    CGFloat _itemY[3];
    int _itemCount;
    NSRect _exampleJPRect, _exampleKanaRect, _exampleENRect;
    NSRect _divider2Rect;
    NSRect _prevButtonRect, _nextButtonRect, _modeButtonRect, _copyButtonRect;
    NSRect _hintRect;
    CGFloat _minHeight;
    BOOL _embedded;   // yes when drawn inside an NSPopover (no card chrome)
    CGFloat _margin;  // 0 when embedded, kShadowMargin for the standalone PNG
}

- (void)setCardJSON:(NSDictionary *)dict;
- (void)setMinHeight:(CGFloat)height;
- (void)setEmbedded:(BOOL)embedded;
- (void)relayout;
- (CGFloat)cardHeight;
- (NSDictionary *)cardDictionary;
- (NSString *)modeButtonTitle;
@end

@implementation KBCardView

- (BOOL)isFlipped {
    return YES;
}

- (void)setCardJSON:(NSDictionary *)dict {
    if (_card == dict) return;
    [dict retain];
    [_card release];
    _card = dict;
    _isKanji = [Str(_card, @"modeKind") isEqualToString:@"kanji"];
    _itemCount = 0;
    NSArray *items = _card[@"items"];
    if ([items isKindOfClass:[NSArray class]]) {
        _itemCount = (int)MIN(items.count, 3);
    }
    [self relayout];
    [self setNeedsDisplay:YES];
}

- (void)setMinHeight:(CGFloat)height {
    if (_minHeight == height) return;
    _minHeight = height;
    [self relayout];
    [self setNeedsDisplay:YES];
}

- (void)setEmbedded:(BOOL)embedded {
    if (_embedded == embedded) return;
    _embedded = embedded;
    [self relayout];
    [self setNeedsDisplay:YES];
}

- (NSDictionary *)cardDictionary {
    return _card;
}

- (CGFloat)cardHeight {
    return _height;
}

// Measures where each block goes, in flipped coordinates (top-left origin).
- (void)relayout {
    _margin = _embedded ? 0.0 : kShadowMargin;
    CGFloat innerX = _margin + kPadding;
    CGFloat y = _margin + kPadding;
    NSString *headword = Str(_card, @"headword");

    // Header row: mode pill on the left, JLPT badge on the right.
    NSAttributedString *modeA = Text(Str(_card, @"modeLabel"), SysFont(9.5, NSFontWeightSemibold),
                                     [NSColor controlAccentColor], NSTextAlignmentCenter, 1.4);
    CGFloat modeW = TextWidth(modeA) + 18.0;
    _modePill = NSMakeRect(innerX, y, modeW, kPillHeight);

    NSAttributedString *badgeA = Text(Str(_card, @"badge"), SysFont(10.5, NSFontWeightBold),
                                      [NSColor whiteColor], NSTextAlignmentCenter, 0.6);
    CGFloat badgeW = TextWidth(badgeA) + 16.0;
    _badgePill = NSMakeRect(innerX + kContentWidth - badgeW, y, badgeW, kPillHeight);

    y += kPillHeight + 18.0;

    // Headword.
    CGFloat headwordSize = _isKanji ? 58.0 : 40.0;
    NSAttributedString *hwA = Text(headword, SysFont(headwordSize, NSFontWeightSemibold),
                                   [NSColor labelColor], NSTextAlignmentCenter, 0);
    CGFloat hwH = TextHeight(hwA, kContentWidth);
    _headwordRect = NSMakeRect(innerX, y, kContentWidth, hwH);
    y = NSMaxY(_headwordRect) + 4.0;

    // Small meta line under the headword (stroke count or part of speech).
    NSString *meta = Str(_card, @"meta");
    if (meta.length > 0) {
        NSAttributedString *metaA = Text(meta, SysFont(10.5, NSFontWeightMedium),
                                         [NSColor tertiaryLabelColor], NSTextAlignmentCenter, 0.2);
        CGFloat mh = TextHeight(metaA, kContentWidth);
        _metaRect = NSMakeRect(innerX, y, kContentWidth, mh);
        y = NSMaxY(_metaRect) + 8.0;
    } else {
        _metaRect = NSZeroRect;
        y += 4.0;
    }

    // Reading groups (on'yomi / kun'yomi for kanji, single reading for kotoba).
    NSRect *labelRects[2]  = {&_r1LabelRect, &_r2LabelRect};
    NSRect *valueRects[2]  = {&_r1ValueRect, &_r2ValueRect};
    NSRect *romajiRects[2] = {&_r1RomajiRect, &_r2RomajiRect};
    NSString *labelKeys[2]  = {@"reading1Label", @"reading2Label"};
    NSString *valueKeys[2]  = {@"reading1", @"reading2"};
    NSString *romajiKeys[2] = {@"reading1Romaji", @"reading2Romaji"};

    for (int i = 0; i < 2; i++) {
        NSString *value = Str(_card, valueKeys[i]);
        if (value.length == 0) {
            *labelRects[i] = NSZeroRect;
            *valueRects[i] = NSZeroRect;
            *romajiRects[i] = NSZeroRect;
            continue;
        }
        NSString *label = Str(_card, labelKeys[i]);
        CGFloat gy = y;

        NSAttributedString *la = Text([label uppercaseString], SysFont(9.5, NSFontWeightSemibold),
                                      [NSColor tertiaryLabelColor], NSTextAlignmentCenter, 1.1);
        CGFloat lh = TextHeight(la, kContentWidth);
        *labelRects[i] = NSMakeRect(innerX, gy, kContentWidth, lh);
        gy = NSMaxY(*labelRects[i]) + 2.0;

        NSAttributedString *va = Text(value, SysFont(20.0, NSFontWeightMedium),
                                      [NSColor labelColor], NSTextAlignmentCenter, 0.5);
        CGFloat vh = TextHeight(va, kContentWidth);
        *valueRects[i] = NSMakeRect(innerX, gy, kContentWidth, vh);
        gy = NSMaxY(*valueRects[i]) + 1.0;

        NSString *romaji = Str(_card, romajiKeys[i]);
        if (romaji.length > 0) {
            NSAttributedString *ra = Text(romaji, MonoFont(12.5),
                                          [NSColor secondaryLabelColor], NSTextAlignmentCenter, 0);
            CGFloat rh = TextHeight(ra, kContentWidth);
            *romajiRects[i] = NSMakeRect(innerX, gy, kContentWidth, rh);
            gy = NSMaxY(*romajiRects[i]);
        } else {
            *romajiRects[i] = NSZeroRect;
        }

        y = gy + 14.0;
    }

    // Meaning.
    NSAttributedString *ma = Text(Str(_card, @"meaning"), SysFont(15.5, NSFontWeightRegular),
                                  [NSColor labelColor], NSTextAlignmentCenter, 0.2);
    CGFloat mh = TextHeight(ma, kContentWidth);
    _meaningRect = NSMakeRect(innerX, y, kContentWidth, mh);
    y = NSMaxY(_meaningRect) + 16.0;

    // Divider + section header (only when there is a list or an example).
    BOOL hasItems = _itemCount > 0;
    BOOL hasExample = Str(_card, @"exampleJP").length > 0;
    _divider1Rect = NSMakeRect(innerX + 12, y, kContentWidth - 24, 1);
    y = NSMaxY(_divider1Rect) + 14.0;

    NSString *section = Str(_card, @"section");
    if (section.length > 0 && (hasItems || hasExample)) {
        NSAttributedString *sa = Text([section uppercaseString], SysFont(9.5, NSFontWeightSemibold),
                                      [NSColor tertiaryLabelColor], NSTextAlignmentLeft, 1.2);
        CGFloat sh = TextHeight(sa, kContentWidth);
        _sectionRect = NSMakeRect(innerX, y, kContentWidth, sh);
        y = NSMaxY(_sectionRect) + 8.0;
    } else {
        _sectionRect = NSZeroRect;
    }

    // Compound rows: "日本 にほん" on one line, the gloss below.
    for (int i = 0; i < _itemCount; i++) {
        _itemY[i] = y;
        y += 42.0;
    }

    // Example block.
    if (hasExample) {
        CGFloat ey = y;
        NSAttributedString *jpA = Text(Str(_card, @"exampleJP"), SysFont(14.5, NSFontWeightRegular),
                                       [NSColor labelColor], NSTextAlignmentLeft, 0);
        CGFloat jpH = TextHeight(jpA, kContentWidth);
        _exampleJPRect = NSMakeRect(innerX, ey, kContentWidth, jpH);
        ey = NSMaxY(_exampleJPRect) + 3.0;

        NSString *kana = Str(_card, @"exampleKana");
        if (kana.length > 0) {
            NSAttributedString *kanaA = Text(kana, SysFont(11.5, NSFontWeightRegular),
                                             [NSColor tertiaryLabelColor], NSTextAlignmentLeft, 0);
            CGFloat kh = TextHeight(kanaA, kContentWidth);
            _exampleKanaRect = NSMakeRect(innerX, ey, kContentWidth, kh);
            ey = NSMaxY(_exampleKanaRect) + 4.0;
        } else {
            _exampleKanaRect = NSZeroRect;
        }

        NSAttributedString *enA = Text(Str(_card, @"exampleEN"), SysFont(12.5, NSFontWeightRegular),
                                       [NSColor secondaryLabelColor], NSTextAlignmentLeft, 0);
        CGFloat eh = TextHeight(enA, kContentWidth);
        _exampleENRect = NSMakeRect(innerX, ey, kContentWidth, eh);
        ey = NSMaxY(_exampleENRect);
        y = ey + 14.0;
    } else {
        _exampleJPRect = _exampleKanaRect = _exampleENRect = NSZeroRect;
    }

    // Footer: navigation buttons and a hint line.
    _divider2Rect = NSMakeRect(innerX, y, kContentWidth, 1);
    y = NSMaxY(_divider2Rect) + 14.0;

    CGFloat buttonHeight = 30.0;
    CGFloat bx = innerX;
    _prevButtonRect = NSMakeRect(bx, y, 34, buttonHeight);
    bx += 34 + 8;
    _nextButtonRect = NSMakeRect(bx, y, 34, buttonHeight);
    bx += 34 + 8;

    _modeButtonRect = NSMakeRect(bx, y, 74, buttonHeight);
    bx += 74 + 8;
    _copyButtonRect = NSMakeRect(bx, y, 56, buttonHeight);
    y += buttonHeight;

    NSAttributedString *hintA = Text(@"← → cards   T mode   Esc close", SysFont(10.0, NSFontWeightRegular),
                                     [NSColor tertiaryLabelColor], NSTextAlignmentCenter, 0);
    CGFloat hintH = TextHeight(hintA, kContentWidth);
    _hintRect = NSMakeRect(innerX, y + 8.0, kContentWidth, hintH);
    y = NSMaxY(_hintRect);

    CGFloat naturalHeight = y + kPadding + _margin;

    // Every card is rendered at exactly the same height so that flipping
    // through the decks never resizes or moves the panel. Spare space on the
    // shorter cards is spread across two flexible gaps - one above the
    // example/compound block, one above the footer - instead of collecting in a
    // single hole at the bottom.
    _height = naturalHeight;
    if (_minHeight > naturalHeight) {
        CGFloat extra = _minHeight - naturalHeight;
        CGFloat topShare = round(extra * 0.4);

        // The example/compound block gets its share of the spare space...
        _divider1Rect.origin.y += topShare;
        _sectionRect.origin.y += topShare;
        for (int i = 0; i < _itemCount; i++) _itemY[i] += topShare;
        _exampleJPRect.origin.y += topShare;
        _exampleKanaRect.origin.y += topShare;
        _exampleENRect.origin.y += topShare;

        // ...but the footer is shifted by the FULL remainder, so its position is
        // identical on every card (no jumping nav buttons when content differs).
        _divider2Rect.origin.y += extra;
        _prevButtonRect.origin.y += extra;
        _nextButtonRect.origin.y += extra;
        _modeButtonRect.origin.y += extra;
        _copyButtonRect.origin.y += extra;
        _hintRect.origin.y += extra;

        _height = _minHeight;
    }
    _cardRect = NSMakeRect(_margin, _margin,
                           _embedded ? self.bounds.size.width : kCardWidth,
                           MAX(_height - 2 * _margin, 1));
}

// Stored so drawRect and hit testing agree on the toggle label.
- (NSString *)modeButtonTitle {
    return [Str(_card, @"modeKind") isEqualToString:@"kanji"] ? @"→ 言葉" : @"→ 漢字";
}

- (void)drawRect:(NSRect)dirtyRect {
    if (!_card) return;

    if (_embedded) {
        // The popover's own material is very translucent on current macOS; an
        // opaque face keeps the kanji crisp over whatever is behind the menu
        // bar, while the popover still supplies border, shadow and arrow.
        [[NSColor windowBackgroundColor] setFill];
        NSRectFill(self.bounds);
    } else {
        // Standalone chrome (used for the PNG snapshots).
        NSBezierPath *cardPath = [NSBezierPath bezierPathWithRoundedRect:_cardRect
                                                                 xRadius:14
                                                                 yRadius:14];
        [[[NSColor windowBackgroundColor] colorWithAlphaComponent:0.99] setFill];
        [cardPath fill];
        [[[NSColor separatorColor] colorWithAlphaComponent:0.9] setStroke];
        cardPath.lineWidth = 1.0;
        [cardPath stroke];
    }

    // Header.
    DrawPill(_modePill, [[NSColor controlAccentColor] colorWithAlphaComponent:0.15],
             Str(_card, @"modeLabel"), SysFont(9.5, NSFontWeightSemibold),
             [NSColor controlAccentColor], 1.4);
    DrawPill(_badgePill, [NSColor systemOrangeColor], Str(_card, @"badge"),
             SysFont(10.5, NSFontWeightBold), [NSColor whiteColor], 0.6);

    // Headword + meta.
    CGFloat headwordSize = _isKanji ? 58.0 : 40.0;
    Draw(Text(Str(_card, @"headword"), SysFont(headwordSize, NSFontWeightSemibold),
              [NSColor labelColor], NSTextAlignmentCenter, 0), _headwordRect);
    NSString *meta = Str(_card, @"meta");
    if (meta.length > 0) {
        Draw(Text(meta, SysFont(10.5, NSFontWeightMedium), [NSColor tertiaryLabelColor],
                  NSTextAlignmentCenter, 0.2), _metaRect);
    }

    // Readings.
    NSRect labelRects[2]  = {_r1LabelRect, _r2LabelRect};
    NSRect valueRects[2]  = {_r1ValueRect, _r2ValueRect};
    NSRect romajiRects[2] = {_r1RomajiRect, _r2RomajiRect};
    NSString *labelKeys[2]  = {@"reading1Label", @"reading2Label"};
    NSString *valueKeys[2]  = {@"reading1", @"reading2"};
    NSString *romajiKeys[2] = {@"reading1Romaji", @"reading2Romaji"};
    for (int i = 0; i < 2; i++) {
        NSString *value = Str(_card, valueKeys[i]);
        if (value.length == 0) continue;
        Draw(Text([Str(_card, labelKeys[i]) uppercaseString], SysFont(9.5, NSFontWeightSemibold),
                  [NSColor tertiaryLabelColor], NSTextAlignmentCenter, 1.1), labelRects[i]);
        Draw(Text(value, SysFont(20.0, NSFontWeightMedium), [NSColor labelColor],
                  NSTextAlignmentCenter, 0.5), valueRects[i]);
        NSString *romaji = Str(_card, romajiKeys[i]);
        if (romaji.length > 0) {
            Draw(Text(romaji, MonoFont(12.5), [NSColor secondaryLabelColor],
                      NSTextAlignmentCenter, 0), romajiRects[i]);
        }
    }

    // Meaning.
    Draw(Text(Str(_card, @"meaning"), SysFont(15.5, NSFontWeightRegular), [NSColor labelColor],
              NSTextAlignmentCenter, 0.2), _meaningRect);

    // Divider + section title.
    DrawSeparator(_divider1Rect);
    NSString *section = Str(_card, @"section");
    if (section.length > 0) {
        Draw(Text([section uppercaseString], SysFont(9.5, NSFontWeightSemibold),
                  [NSColor tertiaryLabelColor], NSTextAlignmentLeft, 1.2), _sectionRect);
    }

    // Compound rows.
    NSArray *items = _card[@"items"];
    for (int i = 0; i < _itemCount; i++) {
        NSDictionary *item = items[i];
        CGFloat rowY = _itemY[i];
        NSString *head = Str(item, @"head");
        NSString *sub = Str(item, @"sub");
        NSString *tail = Str(item, @"tail");

        NSAttributedString *headA = Text(head, SysFont(16.0, NSFontWeightSemibold),
                                         [NSColor labelColor], NSTextAlignmentLeft, 0.3);
        CGFloat headW = TextWidth(headA);
        Draw(headA, NSMakeRect(_margin + kPadding, rowY, headW, 20));

        NSAttributedString *subA = Text(sub, SysFont(12.5, NSFontWeightRegular),
                                        [NSColor secondaryLabelColor], NSTextAlignmentLeft, 0);
        CGFloat subW = MIN(TextWidth(subA), kContentWidth - headW - 8);
        Draw(subA, NSMakeRect(_margin + kPadding + headW + 8, rowY + 2, subW, 18));

        NSAttributedString *tailA = Text(tail, SysFont(12.0, NSFontWeightRegular),
                                         [NSColor secondaryLabelColor], NSTextAlignmentLeft, 0);
        Draw(tailA, NSMakeRect(_margin + kPadding, rowY + 21, kContentWidth, 16));
    }

    // Example.
    if (_exampleJPRect.size.height > 0) {
        Draw(Text(Str(_card, @"exampleJP"), SysFont(14.5, NSFontWeightRegular), [NSColor labelColor],
                  NSTextAlignmentLeft, 0), _exampleJPRect);
        NSString *kana = Str(_card, @"exampleKana");
        if (kana.length > 0) {
            Draw(Text(kana, SysFont(11.5, NSFontWeightRegular), [NSColor tertiaryLabelColor],
                      NSTextAlignmentLeft, 0), _exampleKanaRect);
        }
        Draw(Text(Str(_card, @"exampleEN"), SysFont(12.5, NSFontWeightRegular),
                  [NSColor secondaryLabelColor], NSTextAlignmentLeft, 0), _exampleENRect);
    }

    // Footer.
    DrawSeparator(_divider2Rect);
    DrawButton(_prevButtonRect, @"‹", NO);
    DrawButton(_nextButtonRect, @"›", YES);
    DrawButton(_modeButtonRect, [self modeButtonTitle], NO);
    DrawButton(_copyButtonRect, @"Copy", NO);
    Draw(Text(@"← → cards   T mode   Esc close", SysFont(10.0, NSFontWeightRegular),
              [NSColor tertiaryLabelColor], NSTextAlignmentCenter, 0), _hintRect);
}

- (void)mouseDown:(NSEvent *)event {
    NSPoint p = [self convertPoint:event.locationInWindow fromView:nil];
    if (NSPointInRect(p, _prevButtonRect)) { GoPrev(); return; }
    if (NSPointInRect(p, _nextButtonRect)) { GoNext(); return; }
    if (NSPointInRect(p, _modeButtonRect)) { GoToggleMode(); return; }
    if (NSPointInRect(p, _copyButtonRect)) { CopyCardToPasteboard(_card); return; }
}

@end

#pragma mark - Delegates

@interface KBDelegate : NSObject <NSMenuDelegate, NSPopoverDelegate>
- (void)statusItemClicked:(id)sender;
- (void)menuNext:(id)sender;
- (void)menuPrev:(id)sender;
- (void)menuToggleMode:(id)sender;
- (void)menuToggleAuto:(id)sender;
- (void)menuToggleRomaji:(id)sender;
- (void)menuToggleFurigana:(id)sender;
- (void)menuCopy:(id)sender;
- (void)menuQuit:(id)sender;
@end

#pragma mark - Globals

static NSStatusItem *gStatus = nil;
static NSPopover *gPopover = nil;
static KBCardView *gCard = nil;
static KBDelegate *gDelegate = nil;
static NSMenu *gMenu = nil;
static id gMonitor = nil;
static id gClickMonitor = nil;
static NSMenuItem *gAutoItem = nil;
static NSMenuItem *gRomajiItem = nil;
static NSMenuItem *gFuriganaItem = nil;
static BOOL gPopoverShownAtMouseDown = NO;     // was the card open when this click started?
static BOOL gWantedOpen = NO;                  // clicked intent, also survives the close animation
static double gFixedCardHeight = 0;
static volatile int gPanelVisibleFlag = 0;
static BOOL gRestoreFocusOnClose = NO;
static BOOL gClosingPopover = NO;
static NSString *gPendingTitle = nil;
static BOOL gDebugFrame = NO;

// Optional frame/title tracing: KB_DEBUG_FRAME=1 in the environment.
static void DebugFrame(NSString *format, ...) {
    if (!gDebugFrame) return;
    va_list args;
    va_start(args, format);
    NSString *line = [[NSString alloc] initWithFormat:format arguments:args];
    va_end(args);
    fprintf(stderr, "KBDBG %.3f %s\n", NSProcessInfo.processInfo.systemUptime, line.UTF8String);
    [line release];
}

#pragma mark - Popover

// Height of the card inside the popover: the measured fixed card height minus
// the transparent shadow margin the standalone renderer uses.
static void ApplyStatusTitle(NSString *text) {
    gStatus.button.title = text;
    gStatus.button.toolTip = text;
}

// Changing the status item title re-lays-out the menu bar item while an
// NSPopover is anchored to it, which makes the popover jump or dismiss. While
// the card is open we freeze the title and apply the latest one on close.
static void SetStatusTitle(NSString *text) {
    if (gPopover.isShown) {
        [gPendingTitle release];
        gPendingTitle = [text copy];
        return;
    }
    ApplyStatusTitle(text);
}

static CGFloat EmbeddedContentHeight(void) {
    return MAX(gFixedCardHeight - 2.0 * kShadowMargin, 200.0);
}

// One click = one action. The click records what the user wants; a click that
// lands while the popover is still closing is remembered and honored as soon as
// the close finishes, so reopening never takes two clicks.
static void HandleStatusItemClick(void) {
    if (gClosingPopover) {
        gWantedOpen = YES;
        if (!gPopover.isShown) ShowPopover(); // close already finished
        return;
    }
    BOOL wasShown = gPopoverShownAtMouseDown || gPopover.isShown;
    gPopoverShownAtMouseDown = NO;
    gWantedOpen = !wasShown;
    if (gWantedOpen) {
        ShowPopover();
    } else if (gPopover.isShown) {
        HidePopover(YES);
    }
}

static void ShowPopover(void) {
    if (!gPopover || !gStatus.button) return;
    DebugFrame(@"popover show");
    gWantedOpen = YES;
    [NSApp activateIgnoringOtherApps:YES];
    // NSPopover animates itself (grows out of the item, arrow included) and
    // dismisses on an outside click thanks to the transient behavior.
    gClosingPopover = NO;
    [gPopover showRelativeToRect:gStatus.button.bounds
                          ofView:gStatus.button
                  preferredEdge:NSRectEdgeMinY];
    gPanelVisibleFlag = gPopover.isShown ? 1 : 0;
}

static void HidePopover(BOOL restoreFocus) {
    if (!gPopover || !gPopover.isShown) return;
    DebugFrame(@"popover hide");
    gWantedOpen = NO;
    gClosingPopover = YES;
    gRestoreFocusOnClose = restoreFocus;
    // Close immediately. A custom fade here would keep the popover in a
    // "closing" state for a few hundred milliseconds, during which clicks were
    // ignored (that is what made a quick reopen need two clicks).
    [gPopover close];
}

#pragma mark - Keyboard

static NSEvent *HandleKey(NSEvent *event) {
    if (!gPopover || !gPopover.isShown) return event;
    switch (event.keyCode) {
        case 123: GoPrev(); return nil;   // left arrow
        case 124: GoNext(); return nil;   // right arrow
        case 49:  GoNext(); return nil;   // space
        case 53:  HidePopover(YES); return nil; // escape
        default: break;
    }
    NSString *chars = [event.charactersIgnoringModifiers lowercaseString];
    if ([chars isEqualToString:@"t"]) { GoToggleMode(); return nil; }
    if ([chars isEqualToString:@"a"]) { GoToggleAuto(); return nil; }
    if ([chars isEqualToString:@"r"]) { GoToggleRomaji(); return nil; }
    if ([chars isEqualToString:@"f"]) { GoToggleFurigana(); return nil; }
    if ([chars isEqualToString:@"c"]) { CopyCardToPasteboard([gCard cardDictionary]); return nil; }
    if ([chars isEqualToString:@"q"]) { [NSApp terminate:nil]; return nil; }
    return event;
}

#pragma mark - Menu

static NSMenuItem *AddMenuItem(NSMenu *menu, NSString *title, SEL action, NSString *key) {
    NSMenuItem *item = [[[NSMenuItem alloc] initWithTitle:title action:action keyEquivalent:(key ?: @"")] autorelease];
    [item setTarget:gDelegate];
    [menu addItem:item];
    return item;
}

static void BuildMenu(void) {
    gMenu = [[NSMenu alloc] initWithTitle:@"KanjiBar"];
    gMenu.autoenablesItems = NO;
    AddMenuItem(gMenu, @"Next Card", @selector(menuNext:), nil);
    AddMenuItem(gMenu, @"Previous Card", @selector(menuPrev:), nil);
    AddMenuItem(gMenu, @"Toggle Kanji / Kotoba", @selector(menuToggleMode:), @"t");
    [gMenu addItem:[NSMenuItem separatorItem]];
    gAutoItem = AddMenuItem(gMenu, @"Auto-switch 漢字/言葉 every 15 s", @selector(menuToggleAuto:), @"a");
    gFuriganaItem = AddMenuItem(gMenu, @"Show Furigana in Menu Bar", @selector(menuToggleFurigana:), @"f");
    gRomajiItem = AddMenuItem(gMenu, @"Show Romaji in Menu Bar", @selector(menuToggleRomaji:), @"r");
    [gMenu addItem:[NSMenuItem separatorItem]];
    AddMenuItem(gMenu, @"Copy Current Card", @selector(menuCopy:), @"c");
    [gMenu addItem:[NSMenuItem separatorItem]];
    AddMenuItem(gMenu, @"Quit KanjiBar", @selector(menuQuit:), @"q");
}

#pragma mark - Window delegate

@implementation KBDelegate

- (void)statusItemClicked:(id)sender {
    NSEvent *event = [NSApp currentEvent];
    if (event.type == NSEventTypeRightMouseUp) {
        [NSMenu popUpContextMenu:gMenu withEvent:event forView:sender];
        return;
    }
    HandleStatusItemClick();
}

- (void)menuNext:(id)sender { GoNext(); }
- (void)menuPrev:(id)sender { GoPrev(); }
- (void)menuToggleMode:(id)sender { GoToggleMode(); }
- (void)menuToggleAuto:(id)sender { GoToggleAuto(); }
- (void)menuToggleRomaji:(id)sender { GoToggleRomaji(); }
- (void)menuToggleFurigana:(id)sender { GoToggleFurigana(); }
- (void)menuCopy:(id)sender { CopyCardToPasteboard([gCard cardDictionary]); }
- (void)menuQuit:(id)sender { [NSApp terminate:nil]; }

- (void)popoverWillShow:(NSNotification *)notification {
    gPanelVisibleFlag = 1;
}

- (void)popoverDidClose:(NSNotification *)notification {
    gPanelVisibleFlag = 0;
    BOOL closedByUs = gClosingPopover;
    gClosingPopover = NO;
    DebugFrame(@"popover closed");
    if (gPendingTitle) {
        ApplyStatusTitle(gPendingTitle);
        [gPendingTitle release];
        gPendingTitle = nil;
    }
    if (!closedByUs) gWantedOpen = NO; // dismissed by an outside click
    if (gWantedOpen) {
        // A click arrived while the card was closing: honor it now.
        ShowPopover();
        return;
    }
    if (gRestoreFocusOnClose) {
        gRestoreFocusOnClose = NO;
        [NSApp hide:nil]; // hand focus back to the previously active app
    }
}

@end

#pragma mark - Entry points

void KBStart(void) {
    @autoreleasepool {
        [NSApplication sharedApplication];
        [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
        [NSApp finishLaunching];

        gDelegate = [[KBDelegate alloc] init];

        gStatus = [[[NSStatusBar systemStatusBar] statusItemWithLength:NSVariableStatusItemLength] retain];
        gStatus.button.title = @"漢";
        gStatus.button.target = gDelegate;
        gStatus.button.action = @selector(statusItemClicked:);
        [gStatus.button sendActionOn:(NSEventMaskLeftMouseUp | NSEventMaskRightMouseUp)];
        [gStatus.button setFont:SysFont(13.0, NSFontWeightMedium)];
        gStatus.button.lineBreakMode = NSLineBreakByTruncatingTail;
        gStatus.button.toolTip = @"KanjiBar — click for a kanji / kotoba card";

        BuildMenu();

        gCard = [[KBCardView alloc] initWithFrame:NSMakeRect(0, 0, kCardWidth, 561)];
        [gCard setEmbedded:YES];

        NSViewController *vc = [[NSViewController alloc] init];
        vc.view = gCard;

        gPopover = [[NSPopover alloc] init];
        gPopover.behavior = NSPopoverBehaviorTransient;
        gPopover.animates = YES;
        gPopover.delegate = gDelegate;
        gPopover.contentViewController = vc;
        gPopover.contentSize = NSMakeSize(kCardWidth, 561);

        gDebugFrame = getenv("KB_DEBUG_FRAME") != NULL;

        gMonitor = [[NSEvent addLocalMonitorForEventsMatchingMask:NSEventMaskKeyDown
                                                           handler:^NSEvent *(NSEvent *event) {
            return HandleKey(event);
        }] retain];

        gClickMonitor = [[NSEvent addLocalMonitorForEventsMatchingMask:(NSEventMaskLeftMouseDown |
                                                                   NSEventMaskRightMouseDown)
                                                                handler:^NSEvent *(NSEvent *event) {
            gPopoverShownAtMouseDown = gPopover.isShown && !gClosingPopover;
            return event;
        }] retain];
    }
}

void KBRun(void) {
    [NSApp run];
}

void KBShowNow(void) {
    dispatch_after(dispatch_time(DISPATCH_TIME_NOW, (int64_t)(1.0 * NSEC_PER_SEC)),
                   dispatch_get_main_queue(), ^{
        ShowPopover();
    });
}

void KBHideNow(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        HidePopover(YES);
    });
}

void KBQuit(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        [NSApp terminate:nil];
    });
}

void KBShowCard(const char *json) {
    if (json == NULL) return;
    NSString *payload = [[NSString alloc] initWithUTF8String:json];
    dispatch_async(dispatch_get_main_queue(), ^{
        @autoreleasepool {
            NSDictionary *card = nil;
            NSData *data = [payload dataUsingEncoding:NSUTF8StringEncoding];
            if (data) {
                id parsed = [NSJSONSerialization JSONObjectWithData:data options:0 error:NULL];
                if ([parsed isKindOfClass:[NSDictionary class]]) card = parsed;
            }
            if (card && gCard) {
                [gCard setCardJSON:card];
                [gCard setMinHeight:gFixedCardHeight > 0 ? EmbeddedContentHeight() : 0];
                CGFloat height = [gCard cardHeight];
                [gCard setFrameSize:NSMakeSize(kCardWidth, height)];
                NSSize desired = NSMakeSize(kCardWidth, height);
                if (!NSEqualSizes(gPopover.contentSize, desired)) {
                    gPopover.contentSize = desired;
                }
                [gCard setNeedsDisplay:YES];
                DebugFrame(@"card content=%.0fx%.0f shown=%d", kCardWidth, height, gPopover.isShown);
            }
            [payload release];
        }
    });
}

void KBSetTitle(const char *title) {
    if (title == NULL) return;
    NSString *text = [[NSString alloc] initWithUTF8String:title];
    dispatch_async(dispatch_get_main_queue(), ^{
        @autoreleasepool {
            DebugFrame(@"title=%s", text.UTF8String);
            SetStatusTitle(text);
            [text release];
        }
    });
}

void KBSetMenuState(int autoOn, int romajiOn, int furiganaOn) {
    dispatch_async(dispatch_get_main_queue(), ^{
        @autoreleasepool {
            if (gAutoItem) gAutoItem.state = autoOn ? NSControlStateValueOn : NSControlStateValueOff;
            if (gRomajiItem) gRomajiItem.state = romajiOn ? NSControlStateValueOn : NSControlStateValueOff;
            if (gFuriganaItem) gFuriganaItem.state = furiganaOn ? NSControlStateValueOn : NSControlStateValueOff;
        }
    });
}

void KBSnapshot(const char *json, const char *path, int dark) {
    @autoreleasepool {
        [NSApplication sharedApplication];
        [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];

        NSString *payload = [[NSString alloc] initWithUTF8String:json];
        NSData *data = [payload dataUsingEncoding:NSUTF8StringEncoding];
        [payload release];
        NSDictionary *card = nil;
        if (data) {
            id parsed = [NSJSONSerialization JSONObjectWithData:data options:0 error:NULL];
            if ([parsed isKindOfClass:[NSDictionary class]]) card = parsed;
        }
        if (!card) {
            fprintf(stderr, "KBSnapshot: invalid card JSON\n");
            return;
        }

        KBCardView *view = [[KBCardView alloc] initWithFrame:NSMakeRect(0, 0, kViewWidth, 400)];
        if (dark) {
            view.appearance = [NSAppearance appearanceNamed:NSAppearanceNameDarkAqua];
        }
        [view setCardJSON:card];
        CGFloat height = [view cardHeight];
        [view setFrameSize:NSMakeSize(kViewWidth, height)];

        // Put the view in an offscreen window so it gets a retina backing store.
        NSWindow *window = [[NSWindow alloc] initWithContentRect:NSMakeRect(-20000, -20000, kViewWidth, height)
                                                       styleMask:NSWindowStyleMaskBorderless
                                                         backing:NSBackingStoreBuffered
                                                           defer:NO];
        window.backgroundColor = [NSColor clearColor];
        window.opaque = NO;
        window.contentView = view;
        [window orderFrontRegardless];
        [view displayIfNeeded];

        NSBitmapImageRep *rep = [view bitmapImageRepForCachingDisplayInRect:view.bounds];
        [view cacheDisplayInRect:view.bounds toBitmapImageRep:rep];
        NSData *png = [rep representationUsingType:NSBitmapImageFileTypePNG properties:@{}];
        NSString *outPath = [NSString stringWithUTF8String:path];
        BOOL ok = png != nil && [png writeToFile:outPath atomically:YES];
        fprintf(stderr, "KBSnapshot: %s -> %s (%ldx%ld)\n",
                ok ? "ok" : "FAILED", path,
                (long)rep.pixelsWide, (long)rep.pixelsHigh);

        [window orderOut:nil];
        [window release];
        [view release];
    }
}

double KBCardNaturalHeight(const char *json) {
    @autoreleasepool {
        [NSApplication sharedApplication];
        if (json == NULL) return 0;
        NSString *payload = [[NSString alloc] initWithUTF8String:json];
        NSData *data = [payload dataUsingEncoding:NSUTF8StringEncoding];
        [payload release];
        NSDictionary *card = nil;
        if (data) {
            id parsed = [NSJSONSerialization JSONObjectWithData:data options:0 error:NULL];
            if ([parsed isKindOfClass:[NSDictionary class]]) card = parsed;
        }
        if (!card) return 0;

        KBCardView *view = [[KBCardView alloc] initWithFrame:NSMakeRect(0, 0, kViewWidth, 400)];
        [view setCardJSON:card];
        CGFloat height = [view cardHeight];
        [view release];
        return (double)height;
    }
}

void KBSetFixedCardHeight(double height) {
    gFixedCardHeight = height;
    if (!gCard) return;
    CGFloat contentHeight = EmbeddedContentHeight();
    [gCard setMinHeight:contentHeight];
    CGFloat h = [gCard cardHeight];
    [gCard setFrameSize:NSMakeSize(kCardWidth, h)];
    gPopover.contentSize = NSMakeSize(kCardWidth, h);
}

int KBIsPanelVisible(void) {
    return gPanelVisibleFlag;
}
