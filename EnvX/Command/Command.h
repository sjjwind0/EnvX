//
//  command.h
//  EnvX
//
//  Created by sjjwind on 07/12/2016.
//  Copyright © 2016 sjjwind. All rights reserved.
//

#import <Cocoa/Cocoa.h>

@interface Command : NSObject

- (NSString *)runCommand:(NSString *)commandName arguments:(NSArray *)args;

@end
