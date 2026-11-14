//go:build darwin

package desktop

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework GameController -framework CoreHaptics
#import <GameController/GameController.h>
#import <CoreHaptics/CoreHaptics.h>

enum { emberOther, emberDualShock4, emberDualSense, emberXbox };

// EmberRumbler drives one controller's motors: an engine per grip when the
// pad has two, as the DualShock 4, the DualSense and Xbox pads do, or one
// for the whole pad.
@interface EmberRumbler : NSObject
@property(retain) GCController *controller;
@property(retain) CHHapticEngine *left, *right;
@property(retain) id<CHHapticPatternPlayer> leftPlayer, rightPlayer;
@end

@implementation EmberRumbler
- (void)dealloc {
	[self stop];
	self.controller = nil;
	self.left = nil;
	self.right = nil;
	[super dealloc];
}

- (void)stop {
	[self.leftPlayer stopAtTime:CHHapticTimeImmediate error:nil];
	[self.rightPlayer stopAtTime:CHHapticTimeImmediate error:nil];
	self.leftPlayer = nil;
	self.rightPlayer = nil;
}
@end

static NSMutableArray<EmberRumbler *> *emberRumblers;

static int emberFamily(GCController *c) {
	NSString *kind = c.productCategory;
	if ([kind containsString:@"DualShock"]) {
		return emberDualShock4;
	}
	if ([kind containsString:@"DualSense"]) {
		return emberDualSense;
	}
	if ([kind containsString:@"Xbox"]) {
		return emberXbox;
	}
	return emberOther;
}

static CHHapticEngine *emberEngine(GCController *c, GCHapticsLocality locality) {
	if (![c.haptics.supportedLocalities containsObject:locality]) {
		return nil;
	}
	CHHapticEngine *e = [c.haptics createEngineWithLocality:locality];
	return [e startAndReturnError:nil] ? e : nil;
}

static EmberRumbler *emberRumbler(int family, int nth) {
	NSArray<GCController *> *pads = [GCController controllers];
	if (emberRumblers == nil) {
		emberRumblers = [[NSMutableArray alloc] init];
	}
	for (NSInteger i = (NSInteger)emberRumblers.count - 1; i >= 0; i--) {
		if (![pads containsObject:emberRumblers[i].controller]) {
			[emberRumblers removeObjectAtIndex:i];
		}
	}
	for (GCController *c in pads) {
		if (emberFamily(c) != family || nth-- != 0) {
			continue;
		}
		for (EmberRumbler *r in emberRumblers) {
			if (r.controller == c) {
				return r;
			}
		}
		if (c.haptics == nil) {
			return nil;
		}
		EmberRumbler *r = [[[EmberRumbler alloc] init] autorelease];
		r.controller = c;
		r.left = emberEngine(c, GCHapticsLocalityLeftHandle);
		r.right = emberEngine(c, GCHapticsLocalityRightHandle);
		if (r.left == nil || r.right == nil) {
			r.left = emberEngine(c, GCHapticsLocalityDefault);
			r.right = nil;
		}
		if (r.left == nil) {
			return nil;
		}
		[emberRumblers addObject:r];
		return r;
	}
	return nil;
}

static id<CHHapticPatternPlayer> emberPlay(CHHapticEngine *e, float strength, double seconds) {
	if (e == nil || strength <= 0) {
		return nil;
	}
	NSArray *params = @[
		[[[CHHapticEventParameter alloc]
			initWithParameterID:CHHapticEventParameterIDHapticIntensity value:strength] autorelease],
		[[[CHHapticEventParameter alloc] initWithParameterID:CHHapticEventParameterIDHapticSharpness value:0.5] autorelease],
	];
	CHHapticEvent *ev = [[[CHHapticEvent alloc] initWithEventType:CHHapticEventTypeHapticContinuous
		parameters:params relativeTime:0 duration:seconds] autorelease];
	CHHapticPattern *pattern = [[[CHHapticPattern alloc] initWithEvents:@[ev] parameters:@[] error:nil] autorelease];
	id<CHHapticPatternPlayer> p = pattern == nil ? nil : [e createPlayerWithPattern:pattern error:nil];
	if (p == nil) {
		return nil;
	}
	if (![p startAtTime:CHHapticTimeImmediate error:nil]) {
		// The engine stops when the app loses focus or the pad sleeps.
		if (![e startAndReturnError:nil] || ![p startAtTime:CHHapticTimeImmediate error:nil]) {
			return nil;
		}
	}
	return p;
}

static void emberRumble(int family, int nth, float low, float high, double seconds) {
	@autoreleasepool {
		EmberRumbler *r = emberRumbler(family, nth);
		if (r == nil) {
			return;
		}
		[r stop];
		if (seconds <= 0) {
			return;
		}
		if (r.right == nil) {
			r.leftPlayer = emberPlay(r.left, MAX(low, high), seconds);
			return;
		}
		r.leftPlayer = emberPlay(r.left, low, seconds);
		r.rightPlayer = emberPlay(r.right, high, seconds);
	}
}

static void emberStopRumbles(void) {
	@autoreleasepool {
		[emberRumblers release];
		emberRumblers = nil;
	}
}

// The C side ends here.
*/
import "C"

import "github.com/lantervale/ember/internal/inputfeed"

// playRumble plays r on the nth connected pad of family through the
// GameController framework's haptics (macOS 11 and later).
func playRumble(family, nth int, r inputfeed.Rumble) {
	C.emberRumble(C.int(family), C.int(nth), C.float(r.Low), C.float(r.High), C.double(r.Seconds))
}

// stopRumbles stops every pad, as the game quits.
func stopRumbles() { C.emberStopRumbles() }
