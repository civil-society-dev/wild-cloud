import { Settings, RotateCcw, AlertCircle } from 'lucide-react';
import { useDnsmasq, useMessages } from '../hooks';
import { Message } from './Message';
import { Card, CardHeader, CardTitle, CardContent, Button } from './ui';

export const DnsmasqSection = () => {
  const { 
    dnsmasqConfig, 
    generateConfig, 
    isGenerating, 
    generateError,
    restart, 
    isRestarting, 
    restartError,
    restartData 
  } = useDnsmasq();
  const { messages, setMessage } = useMessages();

  // Handle success/error messaging
  if (generateError) {
    setMessage('dnsmasq', `Failed to generate dnsmasq config: ${generateError.message}`, 'error');
  } else if (dnsmasqConfig) {
    setMessage('dnsmasq', 'Dnsmasq config generated successfully', 'success');
  }

  if (restartError) {
    setMessage('dnsmasq', `Failed to restart dnsmasq: ${restartError.message}`, 'error');
  } else if (restartData) {
    setMessage('dnsmasq', `Dnsmasq restart: ${restartData.status}`, 'success');
  }
  return (
    <Card>
      <CardHeader>
        <CardTitle>DNS/DHCP Management</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex gap-2">
          <Button onClick={() => generateConfig()} disabled={isGenerating} variant="outline">
            <Settings className="mr-2 h-4 w-4" />
            {isGenerating ? 'Generating...' : 'Generate Dnsmasq Config'}
          </Button>
          <Button onClick={() => restart()} disabled={isRestarting} variant="outline">
            <RotateCcw className={`mr-2 h-4 w-4 ${isRestarting ? 'animate-spin' : ''}`} />
            {isRestarting ? 'Restarting...' : 'Restart Dnsmasq'}
          </Button>
        </div>
        
        {generateError && (
          <div className="p-3 bg-red-50 dark:bg-red-950 border border-red-200 dark:border-red-800 rounded-md flex items-start gap-2">
            <AlertCircle className="h-4 w-4 text-red-600 mt-0.5" />
            <div>
              <p className="text-sm font-medium text-red-800 dark:text-red-200">Generation Error</p>
              <p className="text-sm text-red-700 dark:text-red-300">{generateError.message}</p>
            </div>
          </div>
        )}

        {restartError && (
          <div className="p-3 bg-red-50 dark:bg-red-950 border border-red-200 dark:border-red-800 rounded-md flex items-start gap-2">
            <AlertCircle className="h-4 w-4 text-red-600 mt-0.5" />
            <div>
              <p className="text-sm font-medium text-red-800 dark:text-red-200">Restart Error</p>
              <p className="text-sm text-red-700 dark:text-red-300">{restartError.message}</p>
            </div>
          </div>
        )}

        {restartData && (
          <div className="p-3 bg-green-50 dark:bg-green-950 border border-green-200 dark:border-green-800 rounded-md">
            <p className="text-sm text-green-800 dark:text-green-200">
              ✓ Dnsmasq restart: {restartData.status}
            </p>
          </div>
        )}
        
        <Message message={messages.dnsmasq} />
        
        {dnsmasqConfig && (
          <pre className="p-4 bg-muted rounded-md text-sm overflow-auto max-h-96">
            {dnsmasqConfig}
          </pre>
        )}
      </CardContent>
    </Card>
  );
};