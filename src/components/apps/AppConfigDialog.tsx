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
  existingConfig?: Record<string, any>;
  onSave: (config: Record<string, any>) => void;
  isSaving?: boolean;
}

// Utility function to flatten nested objects with dot notation
function flattenObject(obj: Record<string, any>, prefix = ''): Record<string, string> {
  const flattened: Record<string, string> = {};

  for (const [key, value] of Object.entries(obj)) {
    const newKey = prefix ? `${prefix}.${key}` : key;

    if (value !== null && typeof value === 'object' && !Array.isArray(value)) {
      // Recursively flatten nested objects
      Object.assign(flattened, flattenObject(value, newKey));
    } else {
      // Convert primitive values to strings
      flattened[newKey] = String(value ?? '');
    }
  }

  return flattened;
}

// Utility function to unflatten dot notation back to nested objects
function unflattenObject(obj: Record<string, string>): Record<string, any> {
  const unflattened: Record<string, any> = {};

  for (const [key, value] of Object.entries(obj)) {
    const keys = key.split('.');
    let current = unflattened;

    for (let i = 0; i < keys.length - 1; i++) {
      const k = keys[i];
      if (!(k in current)) {
        current[k] = {};
      }
      current = current[k];
    }

    // Set the final value
    current[keys[keys.length - 1]] = value;
  }

  return unflattened;
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
      let initialConfig: Record<string, string> = {};

      // Debug logging to diagnose the issue
      console.log('[AppConfigDialog] App data:', {
        name: app.name,
        hasDefaultConfig: !!app.defaultConfig,
        defaultConfigKeys: app.defaultConfig ? Object.keys(app.defaultConfig) : [],
        hasExistingConfig: !!existingConfig,
        existingConfigKeys: existingConfig ? Object.keys(existingConfig) : [],
      });

      // Start with default config - flatten nested objects
      if (app.defaultConfig) {
        initialConfig = flattenObject(app.defaultConfig);
      }

      // Override with existing config if provided - also flatten
      if (existingConfig) {
        const flattenedExisting = flattenObject(existingConfig);
        Object.entries(flattenedExisting).forEach(([key, value]) => {
          initialConfig[key] = value;
        });
      }

      setConfig(initialConfig);
    }
  }, [app, existingConfig, open]);

  const handleSave = () => {
    // Unflatten the config back to nested objects before saving
    const unflattenedConfig = unflattenObject(config);
    onSave(unflattenedConfig);
  };

  const handleChange = (key: string, value: string) => {
    setConfig(prev => ({ ...prev, [key]: value }));
  };

  // Convert snake_case and dot notation to Title Case for labels
  const formatLabel = (key: string): string => {
    return key;
  };

  if (!app) return null;

  // Use the flattened config keys for display
  const configKeys = Object.keys(config);
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
              const isRequired = app.defaultSecrets?.some(secret =>
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
                    placeholder={config[key] || ''}
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
