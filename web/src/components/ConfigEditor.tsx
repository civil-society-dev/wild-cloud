import { useState, useEffect } from 'react';
import { useParams } from 'react-router-dom';
import { useConfigYaml } from '../hooks';
import { Button, Textarea } from './ui';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "./ui/card";
import { Settings, Save, RotateCcw } from 'lucide-react';
import { cn } from '../lib/utils';

interface ConfigEditorProps {
  className?: string;
}

export function ConfigEditor({ className }: ConfigEditorProps) {
  const { instanceId } = useParams<{ instanceId: string }>();
  const { yamlContent, isLoading, error, isEndpointMissing, updateYaml, refetch } = useConfigYaml(instanceId || '');

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

  const handleReset = () => {
    if (yamlContent) {
      setEditedContent(yamlContent);
      setHasChanges(false);
    }
  };

  return (
    <Card className={cn("flex flex-col", className)}>
      <CardHeader className="shrink-0">
        <CardTitle className="flex items-center gap-2">
          <Settings className="h-5 w-5" />
          Configuration Editor
        </CardTitle>
        <CardDescription>
          Edit the raw YAML configuration file. This provides direct access to all configuration options.
        </CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col flex-1 min-h-0">
        {error && error instanceof Error && error.message && (
          <div className="p-3 mb-4 bg-red-50 dark:bg-red-950 border border-red-200 dark:border-red-800 rounded-md shrink-0">
            <p className="text-sm text-red-800 dark:text-red-200">
              Error: {error.message}
            </p>
          </div>
        )}

        {isEndpointMissing && (
          <div className="p-3 mb-4 bg-orange-50 dark:bg-orange-950 border border-orange-200 dark:border-orange-800 rounded-md shrink-0">
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
          className="font-mono text-sm w-full flex-1 min-h-0 resize-none whitespace-pre overflow-x-auto"
        />

        <div className="flex items-center justify-between mt-4 shrink-0">
          <div>
            {hasChanges && (
              <span className="text-sm text-orange-600 dark:text-orange-400">
                You have unsaved changes
              </span>
            )}
          </div>
          <div className="flex gap-2">
            <Button
              variant="outline"
              onClick={handleReset}
              disabled={!hasChanges || isLoading}
            >
              <RotateCcw className="h-4 w-4 mr-2" />
              Reset
            </Button>
            <Button
              onClick={handleSave}
              disabled={!hasChanges || isLoading || !!isEndpointMissing}
            >
              <Save className="h-4 w-4 mr-2" />
              Save Changes
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
