import { Card } from './ui/card';
import { Button } from './ui/button';
import { Globe, CheckCircle, BookOpen, ExternalLink } from 'lucide-react';

export function DnsComponent() {
  return (
    <div className="space-y-6">
      {/* Educational Intro Section */}
      <Card className="p-6 bg-gradient-to-r from-green-50 to-emerald-50 dark:from-green-950/20 dark:to-emerald-950/20 border-green-200 dark:border-green-800">
        <div className="flex items-start gap-4">
          <div className="p-3 bg-green-100 dark:bg-green-900/30 rounded-lg">
            <BookOpen className="h-6 w-6 text-green-600 dark:text-green-400" />
          </div>
          <div className="flex-1">
            <h3 className="text-lg font-semibold text-green-900 dark:text-green-100 mb-2">
              What is DNS?
            </h3>
            <p className="text-green-800 dark:text-green-200 mb-3 leading-relaxed">
              DNS (Domain Name System) is like the "phone book" of the internet. Instead of remembering complex IP addresses 
              like "192.168.1.100", you can use friendly names like "my-server.local". When you type a name, DNS translates 
              it to the correct IP address so your devices can find each other.
            </p>
            <p className="text-green-700 dark:text-green-300 mb-4 text-sm">
              Your personal cloud runs its own DNS service so devices can easily find services like "photos.home" or "media.local" 
              without needing to remember numbers.
            </p>
            <Button variant="outline" size="sm" className="text-green-700 border-green-300 hover:bg-green-100 dark:text-green-300 dark:border-green-700 dark:hover:bg-green-900/20">
              <ExternalLink className="h-4 w-4 mr-2" />
              Learn more about DNS
            </Button>
          </div>
        </div>
      </Card>

      <Card className="p-6">
        <div className="flex items-center gap-4 mb-6">
          <div className="p-2 bg-primary/10 rounded-lg">
            <Globe className="h-6 w-6 text-primary" />
          </div>
          <div>
            <h2 className="text-2xl font-semibold">DNS Configuration</h2>
            <p className="text-muted-foreground">
              Manage DNS settings and domain resolution
            </p>
          </div>
        </div>

        <div className="space-y-4">
          <div className="flex items-center gap-2">
            <CheckCircle className="h-5 w-5 text-green-500" />
            <span className="text-sm">Local resolution: Active</span>
          </div>
          
          <div className="mt-4">
            <h4 className="font-medium mb-2">DNS Status</h4>
            <p className="text-sm text-muted-foreground">
              DNS service is running and resolving domains correctly.
            </p>
          </div>

          <div className="flex gap-2 justify-end mt-4">
            <Button variant="outline" onClick={() => console.log('Test DNS')}>
              Test DNS
            </Button>
            <Button onClick={() => console.log('Configure DNS')}>
              Configure
            </Button>
          </div>
        </div>
      </Card>
    </div>
  );
}