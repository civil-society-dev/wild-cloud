import { Download, AlertCircle } from 'lucide-react';
import { useAssets, useMessages } from '../hooks';
import { Message } from './Message';
import { Card, CardHeader, CardTitle, CardContent, Button } from './ui';

export const PxeAssetsSection = () => {
  const { downloadAssets, isDownloading, error, data } = useAssets();
  const { messages, setMessage } = useMessages();

  // Handle success/error messaging
  if (error) {
    setMessage('assets', `Failed to download assets: ${error.message}`, 'error');
  } else if (data) {
    setMessage('assets', `PXE Assets: ${data.status}`, 'success');
  }
  return (
    <Card>
      <CardHeader>
        <CardTitle>PXE Boot Assets</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <Button onClick={() => downloadAssets()} disabled={isDownloading} variant="outline">
          <Download className="mr-2 h-4 w-4" />
          {isDownloading ? 'Downloading...' : 'Download/Update PXE Assets'}
        </Button>
        
        {error && (
          <div className="p-3 bg-red-50 dark:bg-red-950 border border-red-200 dark:border-red-800 rounded-md flex items-start gap-2">
            <AlertCircle className="h-4 w-4 text-red-600 mt-0.5" />
            <div>
              <p className="text-sm font-medium text-red-800 dark:text-red-200">Download Error</p>
              <p className="text-sm text-red-700 dark:text-red-300">{error.message}</p>
            </div>
          </div>
        )}

        {data && (
          <div className="p-3 bg-green-50 dark:bg-green-950 border border-green-200 dark:border-green-800 rounded-md">
            <p className="text-sm text-green-800 dark:text-green-200">
              ✓ PXE Assets: {data.status}
            </p>
          </div>
        )}
        
        <Message message={messages.assets} />
      </CardContent>
    </Card>
  );
};