import { useState, useEffect } from 'react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription,
} from '../ui/dialog';
import { Button } from '../ui/button';
import { Input } from '../ui/input';
import { Label } from '../ui/label';
import { Loader2, Info } from 'lucide-react';
import type { App } from '../../services/api';

interface AppConfigDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  app: App | null;
  existingConfig?: Record<string, string>;
  onSave: (config: Record<string, string>) => void;
  isSaving?: boolean;
}

export function AppConfigDialog({
  open,
  onOpenChange,
  app,
  existingConfig,
  onSave,
  isSaving = false,
}: AppConfigDialogProps) {
  const [config, setConfig] = useState<Record<string, string>>({});

  // Initialize config when dialog opens or app changes
  useEffect(() => {
    if (app && open) {
      const initialConfig: Record<string, string> = {};

      // Debug logging to diagnose the issue
      console.log('[AppConfigDialog] App data:', {
        name: app.name,
        hasDefaultConfig: !!app.defaultConfig,
        defaultConfigKeys: app.defaultConfig ? Object.keys(app.defaultConfig) : [],
        hasExistingConfig: !!existingConfig,
        existingConfigKeys: existingConfig ? Object.keys(existingConfig) : [],
      });

      // Start with default config
      if (app.defaultConfig) {
        Object.entries(app.defaultConfig).forEach(([key, value]) => {
          initialConfig[key] = String(value);
        });
      }

      // Override with existing config if provided
      if (existingConfig) {
        Object.entries(existingConfig).forEach(([key, value]) => {
          initialConfig[key] = value;
        });
      }

      setConfig(initialConfig);
    }
  }, [app, existingConfig, open]);

  const handleSave = () => {
    onSave(config);
  };

  const handleChange = (key: string, value: string) => {
    setConfig(prev => ({ ...prev, [key]: value }));
  };

  // Convert snake_case to Title Case for labels
  const formatLabel = (key: string): string => {
    return key
      .split('_')
      .map(word => word.charAt(0).toUpperCase() + word.slice(1))
      .join(' ');
  };

  if (!app) return null;

  const configKeys = Object.keys(app.defaultConfig || {});
  const hasConfig = configKeys.length > 0;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[80vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Configure {app.name}</DialogTitle>
          <DialogDescription>
            {app.description}
          </DialogDescription>
        </DialogHeader>

        {hasConfig ? (
          <div className="space-y-4 py-4">
            {configKeys.map((key) => {
              const isRequired = app.requiredSecrets?.some(secret =>
                secret.toLowerCase().includes(key.toLowerCase())
              );

              return (
                <div key={key} className="space-y-2">
                  <div className="flex items-center gap-2">
                    <Label htmlFor={key}>
                      {formatLabel(key)}
                      {isRequired && <span className="text-red-500">*</span>}
                    </Label>
                    {isRequired && (
                      <span title="Required for secrets generation">
                        <Info className="h-3 w-3 text-muted-foreground" />
                      </span>
                    )}
                  </div>
                  <Input
                    id={key}
                    value={config[key] || ''}
                    onChange={(e) => handleChange(key, e.target.value)}
                    placeholder={String(app.defaultConfig?.[key] || '')}
                    required={isRequired}
                  />
                  {isRequired && (
                    <p className="text-xs text-muted-foreground">
                      This value is used to generate application secrets
                    </p>
                  )}
                </div>
              );
            })}

            {app.dependencies && app.dependencies.length > 0 && (
              <div className="mt-6 p-4 bg-blue-50 dark:bg-blue-950/20 rounded-lg border border-blue-200 dark:border-blue-800">
                <h4 className="text-sm font-medium text-blue-900 dark:text-blue-100 mb-2">
                  Dependencies
                </h4>
                <p className="text-sm text-blue-800 dark:text-blue-200 mb-2">
                  This app requires the following apps to be deployed first:
                </p>
                <ul className="text-sm text-blue-700 dark:text-blue-300 list-disc list-inside">
                  {app.dependencies.map(dep => (
                    <li key={dep}>{dep}</li>
                  ))}
                </ul>
              </div>
            )}
          </div>
        ) : (
          <div className="py-8 text-center text-muted-foreground">
            <p>This app doesn't require any configuration.</p>
            <p className="text-sm mt-2">Click Add to proceed with default settings.</p>
          </div>
        )}

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={isSaving}
          >
            Cancel
          </Button>
          <Button
            onClick={handleSave}
            disabled={isSaving}
          >
            {isSaving ? (
              <>
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                Saving...
              </>
            ) : (
              existingConfig ? 'Update' : 'Add App'
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
