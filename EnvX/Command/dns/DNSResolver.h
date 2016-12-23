//
//  DNSResolver.h
//  EnvX
//
//  Created by sjjwind on 07/12/2016.
//  Copyright © 2016 sjjwind. All rights reserved.
//

#import <Cocoa/Cocoa.h>
#import "Command.h"

@interface DNSResolver : Command

+ (instancetype) shareInstance;

- (BOOL)setDNS:(NSArray<NSString *> *)dnsList;

- (NSArray<NSString *> *)getDNSServerList;

@end
