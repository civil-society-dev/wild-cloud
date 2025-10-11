import { Card } from './ui/card';
import { Button } from './ui/button';
import { Server, Network, Settings, Clock, HelpCircle, CheckCircle, BookOpen, ExternalLink } from 'lucide-react';
import { Input, Label } from './ui';

export function CentralComponent() {
  return (
    <div className="space-y-6">
      {/* Educational Intro Section */}
      <Card className="p-6 bg-gradient-to-r from-blue-50 to-indigo-50 dark:from-blue-950/20 dark:to-indigo-950/20 border-blue-200 dark:border-blue-800">
        <div className="flex items-start gap-4">
          <div className="p-3 bg-blue-100 dark:bg-blue-900/30 rounded-lg">
            <BookOpen className="h-6 w-6 text-blue-600 dark:text-blue-400" />
          </div>
          <div className="flex-1">
            <h3 className="text-lg font-semibold text-blue-900 dark:text-blue-100 mb-2">
              What is the Central Service?
            </h3>
            <p className="text-blue-800 dark:text-blue-200 mb-3 leading-relaxed">
              The Central Service is the "brain" of your personal cloud. It acts as the main coordinator that manages 
              all the different services running on your network. Think of it like the control tower at an airport - 
              it keeps track of what's happening, routes traffic between services, and ensures everything works together smoothly.
            </p>
            <p className="text-blue-700 dark:text-blue-300 mb-4 text-sm">
              This service handles configuration management, service discovery, and provides the web interface you're using right now.
            </p>
            <Button variant="outline" size="sm" className="text-blue-700 border-blue-300 hover:bg-blue-100 dark:text-blue-300 dark:border-blue-700 dark:hover:bg-blue-900/20">
              <ExternalLink className="h-4 w-4 mr-2" />
              Learn more about service orchestration
            </Button>
          </div>
        </div>
      </Card>

      <Card className="p-6">
        <div className="flex items-center gap-4 mb-6">
          <div className="p-2 bg-primary/10 rounded-lg">
            <Server className="h-6 w-6 text-primary" />
          </div>
          <div>
            <h2 className="text-2xl font-semibold">Central Service</h2>
            <p className="text-muted-foreground">
              Monitor and manage the central server service
            </p>
          </div>
        </div>

        <div>
          <h3 className="text-lg font-medium mb-4">Service Status</h3>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-6 mb-6">
            <div className="flex items-center gap-2">
              <Server className="h-5 w-5 text-muted-foreground" />
              <span className="text-sm text-muted-foreground">IP Address: 192.168.8.50</span>
            </div>
            <div className="flex items-center gap-2">
              <Network className="h-5 w-5 text-muted-foreground" />
              <span className="text-sm text-muted-foreground">Network: 192.168.8.0/24</span>
            </div>
            <div className="flex items-center gap-2">
              <Settings className="h-5 w-5 text-muted-foreground" />
              <span className="text-sm text-muted-foreground">Version: 1.0.0 (update available)</span>
            </div>
            <div className="flex items-center gap-2">
              <Clock className="h-5 w-5 text-muted-foreground" />
              <span className="text-sm text-muted-foreground">Age: 12s</span>
            </div>
            <div className="flex items-center gap-2">
              <HelpCircle className="h-5 w-5 text-muted-foreground" />
              <span className="text-sm text-muted-foreground">Platform: ARM</span>
            </div>
            <div className="flex items-center gap-2">
              <CheckCircle className="h-5 w-5 text-green-500" />
              <span className="text-sm text-green-500">File permissions: Good</span>
            </div>
          </div>

          <div className="space-y-4">
            <div>
              <Label htmlFor="ip">IP</Label>
              <div className="flex w-full items-center mt-1">
                <Input id="ip" value="192.168.5.80"/>
                <Button variant="ghost">
                  <HelpCircle/>
                </Button>
              </div>
            </div>
            <div>
              <Label htmlFor="interface">Interface</Label>
              <div className="flex w-full items-center mt-1">
                <Input id="interface" value="eth0"/>
                <Button variant="ghost">
                  <HelpCircle/>
                </Button>
              </div>
            </div>
          </div>

          <div className="flex gap-2 justify-end mt-4">
            <Button onClick={() => console.log('Update service')}>
              Update
            </Button>
            <Button onClick={() => console.log('Restart service')}>
              Restart
            </Button>
            <Button onClick={() => console.log('View log')}>
              View log
            </Button>
          </div>
        </div>
      </Card>
    </div>
  );
}