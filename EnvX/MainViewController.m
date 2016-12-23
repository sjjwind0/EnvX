//
//  ViewController.m
//  EnvX
//
//  Created by sjjwind on 07/12/2016.
//  Copyright © 2016 sjjwind. All rights reserved.
//

#import "MainViewController.h"
#import "Command/dns/DNSResolver.h"

@implementation MainViewController

- (void)viewDidLoad {
  [super viewDidLoad];

  NSArray<NSString *> *dnsList = [[DNSResolver shareInstance] getDNSServerList];
  NSLog(@"dnsList: %@", dnsList);
}


- (void)setRepresentedObject:(id)representedObject {
  [super setRepresentedObject:representedObject];

  // Update the view, if already loaded.
}


@end
