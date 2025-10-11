import { useState, useEffect } from 'react';
import { Settings, Save, X } from 'lucide-react';
import { useConfigYaml } from '../hooks';
import { Button, Textarea } from './ui';
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger} from '@/components/ui/dialog';

export function ConfigEditor() {
  const { yamlContent, isLoading, error, isEndpointMissing, updateYaml, refetch } = useConfigYaml();
  
  const [editedContent, setEditedContent] = useState('');
  const [hasChanges, setHasChanges] = useState(false);

  // Update edited content when YAML content changes
  useEffect(() => {
    if (yamlContent) {
      setEditedContent(yamlContent);
      setHasChanges(false);
    }
  }, [yamlContent]);

  // Track changes
  useEffect(() => {
    setHasChanges(editedContent !== yamlContent);
  }, [editedContent, yamlContent]);

  const handleSave = () => {
    if (!hasChanges) return;
    
    updateYaml(editedContent, {
      onSuccess: () => {
        setHasChanges(false);
      },
      onError: (err) => {
        console.error('Failed to update config:', err);
      }
    });
  };

  const handleOpenChange = (open: boolean) => {
    if (!open && hasChanges) {
      if (!window.confirm('You have unsaved changes. Close anyway?')) {
        return;
      }
    }
    if (open) {
      refetch();
    }
  };

  return (
    <Dialog onOpenChange={handleOpenChange}>
      <DialogTrigger asChild>
        <Button>
          <Settings className="h-4 w-4" />
          Config
        </Button>
      </DialogTrigger>
            
      <DialogContent className="max-w-6xl w-full max-h-[80vh] h-full flex flex-col">
        <DialogHeader>
          <DialogTitle>
            Configuration Editor
          </DialogTitle>
          <DialogDescription>
            Edit the raw YAML configuration file. This provides direct access to all configuration options.
          </DialogDescription>
        </DialogHeader>
        <div className="flex flex-col flex-1">

          {error && error instanceof Error && error.message && (
            <div className="p-3 bg-red-50 dark:bg-red-950 border border-red-200 dark:border-red-800 rounded-md">
              <p className="text-sm text-red-800 dark:text-red-200">
                Error: {error.message}
              </p>
            </div>
          )}
          
          {isEndpointMissing && (
            <div className="p-3 bg-orange-50 dark:bg-orange-950 border border-orange-200 dark:border-orange-800 rounded-md">
              <p className="text-sm text-orange-800 dark:text-orange-200">
                Backend endpoints missing. Raw YAML editing not available.
              </p>
            </div>
          )}
          
          <Textarea
            value={editedContent}
            onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) => setEditedContent(e.target.value)}
            placeholder={isLoading ? "Loading YAML configuration..." : "No configuration found"}
            disabled={isLoading || !!isEndpointMissing}
            className="font-mono text-sm w-full flex-1 min-h-0 resize-none"
          />
          
          {hasChanges && (
            <div className="text-sm text-orange-600 dark:text-orange-400">
              ⚠️ You have unsaved changes
            </div>
          )}
        </div>
        
        <DialogFooter>
          <DialogClose asChild>
            <Button variant="outline">
              Cancel
            </Button>
          </DialogClose>
          <Button 
            onClick={handleSave}
            disabled={!hasChanges || isLoading || !!isEndpointMissing}
          >
            Update Config
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}