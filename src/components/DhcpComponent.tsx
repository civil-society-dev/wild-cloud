import { Card } from './ui/card';
import { Button } from './ui/button';
import { Wifi, HelpCircle, BookOpen, ExternalLink } from 'lucide-react';
import { Input, Label } from './ui';

export function DhcpComponent() {
  return (
    <div className="space-y-6">
      {/* Educational Intro Section */}
      <Card className="p-6 bg-gradient-to-r from-purple-50 to-violet-50 dark:from-purple-950/20 dark:to-violet-950/20 border-purple-200 dark:border-purple-800">
        <div className="flex items-start gap-4">
          <div className="p-3 bg-purple-100 dark:bg-purple-900/30 rounded-lg">
            <BookOpen className="h-6 w-6 text-purple-600 dark:text-purple-400" />
          </div>
          <div className="flex-1">
            <h3 className="text-lg font-semibold text-purple-900 dark:text-purple-100 mb-2">
              What is DHCP?
            </h3>
            <p className="text-purple-800 dark:text-purple-200 mb-3 leading-relaxed">
              DHCP (Dynamic Host Configuration Protocol) is like an automatic "address assignment system" for your network. 
              When a device joins your network, DHCP automatically gives it an IP address, tells it how to connect to the internet, 
              and provides other network settings - no manual configuration needed!
            </p>
            <p className="text-purple-700 dark:text-purple-300 mb-4 text-sm">
              Without DHCP, you'd need to manually assign IP addresses to every device. DHCP makes it so you can just connect 
              a phone, laptop, or smart device and it automatically gets everything it needs to work on your network.
            </p>
            <Button variant="outline" size="sm" className="text-purple-700 border-purple-300 hover:bg-purple-100 dark:text-purple-300 dark:border-purple-700 dark:hover:bg-purple-900/20">
              <ExternalLink className="h-4 w-4 mr-2" />
              Learn more about DHCP
            </Button>
          </div>
        </div>
      </Card>

      <Card className="p-6">
        <div className="flex items-center gap-4 mb-6">
          <div className="p-2 bg-primary/10 rounded-lg">
            <Wifi className="h-6 w-6 text-primary" />
          </div>
          <div>
            <h2 className="text-2xl font-semibold">DHCP Configuration</h2>
            <p className="text-muted-foreground">
              Manage DHCP settings and IP address allocation
            </p>
          </div>
        </div>

        <div className="space-y-4">
          <div className="flex items-center gap-2 mb-4">
            <span className="text-sm font-medium">Status:</span>
            <span className="text-sm text-green-600">Active</span>
          </div>

          <div>
            <Label htmlFor="dhcpRange">IP Range</Label>
            <div className="flex w-full items-center mt-1">
              <Input id="dhcpRange" value="192.168.8.100,192.168.8.239"/>
              <Button variant="ghost">
                <HelpCircle/>
              </Button>
            </div>
          </div>

          <div className="flex gap-2 justify-end mt-4">
            <Button variant="outline" onClick={() => console.log('View DHCP clients')}>
              View Clients
            </Button>
            <Button onClick={() => console.log('Configure DHCP')}>
              Configure
            </Button>
          </div>
        </div>
      </Card>
    </div>
  );
}