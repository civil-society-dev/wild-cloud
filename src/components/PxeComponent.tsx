import { Card } from './ui/card';
import { Button } from './ui/button';
import { HardDrive, BookOpen, ExternalLink } from 'lucide-react';

export function PxeComponent() {
  return (
    <div className="space-y-6">
      {/* Educational Intro Section */}
      <Card className="p-6 bg-gradient-to-r from-orange-50 to-amber-50 dark:from-orange-950/20 dark:to-amber-950/20 border-orange-200 dark:border-orange-800">
        <div className="flex items-start gap-4">
          <div className="p-3 bg-orange-100 dark:bg-orange-900/30 rounded-lg">
            <BookOpen className="h-6 w-6 text-orange-600 dark:text-orange-400" />
          </div>
          <div className="flex-1">
            <h3 className="text-lg font-semibold text-orange-900 dark:text-orange-100 mb-2">
              What is PXE Boot?
            </h3>
            <p className="text-orange-800 dark:text-orange-200 mb-3 leading-relaxed">
              PXE (Preboot Execution Environment) is like having a "network installer" that can set up computers without 
              needing USB drives or DVDs. When you turn on a computer, instead of booting from its hard drive, it can boot 
              from the network and automatically install an operating system or run diagnostics.
            </p>
            <p className="text-orange-700 dark:text-orange-300 mb-4 text-sm">
              This is especially useful for setting up multiple computers in your cloud infrastructure. PXE can automatically 
              install and configure the same operating system on many machines, making it easy to expand your personal cloud.
            </p>
            <Button variant="outline" size="sm" className="text-orange-700 border-orange-300 hover:bg-orange-100 dark:text-orange-300 dark:border-orange-700 dark:hover:bg-orange-900/20">
              <ExternalLink className="h-4 w-4 mr-2" />
              Learn more about network booting
            </Button>
          </div>
        </div>
      </Card>

      <Card className="p-6">
        <div className="flex items-center gap-4 mb-6">
          <div className="p-2 bg-primary/10 rounded-lg">
            <HardDrive className="h-6 w-6 text-primary" />
          </div>
          <div>
            <h2 className="text-2xl font-semibold">PXE Configuration</h2>
            <p className="text-muted-foreground">
              Manage PXE boot assets and network boot configuration
            </p>
          </div>
        </div>

        <div className="space-y-4">
          <div className="flex items-center gap-2 mb-4">
            <span className="text-sm font-medium">Status:</span>
            <span className="text-sm text-green-600">Active</span>
          </div>

          <div>
            <h4 className="font-medium mb-2">Boot Assets</h4>
            <p className="text-sm text-muted-foreground mb-4">
              Manage Talos Linux boot images and iPXE configurations for network booting.
            </p>
          </div>

          <div className="flex gap-2 justify-end mt-4">
            <Button variant="outline" onClick={() => console.log('View assets')}>
              View Assets
            </Button>
            <Button onClick={() => console.log('Download PXE assets')}>
              Download Assets
            </Button>
          </div>
        </div>
      </Card>
    </div>
  );
}