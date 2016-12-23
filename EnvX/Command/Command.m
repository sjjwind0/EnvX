//
//  command.m
//  EnvX
//
//  Created by sjjwind on 07/12/2016.
//  Copyright © 2016 sjjwind. All rights reserved.
//

#import "command.h"

@interface Command ()

@end

@implementation Command

- (NSString *)runCommand:(NSString *)commandName arguments:(NSArray *)args {
  NSTask *task;
  task = [[NSTask alloc] init];
  [task setLaunchPath: @"/bin/bash"];
  
  NSArray *arguments = [NSArray arrayWithObject:commandName];
  arguments = [arguments arrayByAddingObjectsFromArray:args];
  NSString *command = [arguments componentsJoinedByString:@" "];
  [task setArguments: @[@"-c", command]];
  
  NSPipe *pipe;
  pipe = [NSPipe pipe];
  [task setStandardOutput: pipe];
  
  NSFileHandle *file;
  file = [pipe fileHandleForReading];
  
  [task launch];
  
  NSData *data;
  data = [file readDataToEndOfFile];
  return [[NSString alloc] initWithData: data encoding: NSUTF8StringEncoding];  
}

@end
