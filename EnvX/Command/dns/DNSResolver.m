//
//  DNSResolver.m
//  EnvX
//
//  Created by sjjwind on 07/12/2016.
//  Copyright © 2016 sjjwind. All rights reserved.
//

#import "DNSResolver.h"

#import <SystemConfiguration/SystemConfiguration.h>
#import <SecurityFoundation/SecurityFoundation.h>
#import <SecurityFoundation/SFAuthorization.h>

const NSInteger kMaxDNSListSize = 2;

@interface DNSResolver ()

@end

@implementation DNSResolver

+ (instancetype) shareInstance {
  static dispatch_once_t onceToken;
  static DNSResolver *instance;
  dispatch_once(&onceToken, ^{
    instance = [[DNSResolver alloc] init];
  });
  return instance;
}

- (BOOL)setDNS:(NSArray<NSString *> *)dnsList {
  CFStringRef resolvers[kMaxDNSListSize];
  for (int i = 0; i < dnsList.count; i++) {
    resolvers[i] = (__bridge CFStringRef)dnsList[i];
  }
  
  CFIndex dnsCount = dnsList.count;
  
  SCDynamicStoreRef ds = SCDynamicStoreCreate(NULL, CFSTR("setDNS"), NULL, NULL);
  
  CFArrayRef array = CFArrayCreate(NULL, (const void **) resolvers,
                                   dnsCount, &kCFTypeArrayCallBacks);
  
  CFDictionaryRef dict = CFDictionaryCreate(NULL,
      (const void **) (CFStringRef []) { CFSTR("ServerAddresses") },
      (const void **) &array, 1, &kCFTypeDictionaryKeyCallBacks,
      &kCFTypeDictionaryValueCallBacks);
  
  CFArrayRef list = SCDynamicStoreCopyKeyList(ds,
      CFSTR("State:/Network/(Service/.+|Global)/DNS"));
  
  CFIndex i = 0, j = CFArrayGetCount(list);
  if (j <= 0) {
    return FALSE;
  }
  bool ret = TRUE;
  while (i < j) {
    printf("pass\n");
    ret &= SCDynamicStoreSetValue(ds, CFArrayGetValueAtIndex(list, i), dict);
    i++;
  }
  return ret;
}

- (NSArray *)getDNSServerList {
  NSMutableArray *dnsList = [NSMutableArray array];
  NSString *resolvePath = @"/etc/resolv.conf";
  NSString *content = [NSString stringWithContentsOfFile:resolvePath 
                                                encoding:NSUTF8StringEncoding error:nil];
  NSArray<NSString *> *lines = [content componentsSeparatedByString:@"\n"];
  [lines enumerateObjectsUsingBlock:^(NSString *line, NSUInteger idx, BOOL * _Nonnull stop) {
    if ([line hasPrefix:@"#"]) {
      return;
    } else {
      NSArray<NSString *> *resolvers = [line componentsSeparatedByString:@" "];
      if (resolvers.count == 2) {
        if ([resolvers[0] isEqualToString:@"nameserver"]) {
          [dnsList addObject:resolvers[1]];
        }
      }
    }
  }];
  
  return [dnsList copy];
}

@end
