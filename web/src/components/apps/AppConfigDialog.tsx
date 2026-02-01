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
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../ui/select';
import { Loader2, Info, AlertCircle } from 'lucide-react';
import { Alert, AlertDescription } from '../ui/alert';
import type { App } from '../../services/api';
import { useDeployedApps } from '../../hooks/useApps';
import { useInstanceContext } from '../../hooks/useInstanceContext';

interface AppConfigDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  app: App | null;
  existingConfig?: Record<string, unknown>;
  existingAppName?: string; // The current name if editing an existing app
  onSave: (appName: string, config: Record<string, unknown>, requiredAppMappings?: Record<string, string>) => void;
  isSaving?: boolean;
}

// Utility function to flatten nested objects with dot notation
function flattenObject(obj: Record<string, unknown>, prefix = ''): Record<string, string> {
  const flattened: Record<string, string> = {};

  for (const [key, value] of Object.entries(obj)) {
    const newKey = prefix ? `${prefix}.${key}` : key;

    if (value !== null && typeof value === 'object' && !Array.isArray(value)) {
      // Recursively flatten nested objects
      Object.assign(flattened, flattenObject(value as Record<string, unknown>, newKey));
    } else {
      // Convert primitive values to strings
      flattened[newKey] = String(value ?? '');
    }
  }

  return flattened;
}

// Utility function to unflatten dot notation back to nested objects
function unflattenObject(obj: Record<string, string>): Record<string, unknown> {
  const unflattened: Record<string, unknown> = {};

  for (const [key, value] of Object.entries(obj)) {
    const keys = key.split('.');
    let current = unflattened;

    for (let i = 0; i < keys.length - 1; i++) {
      const k = keys[i];
      if (!(k in current)) {
        (current as Record<string, unknown>)[k] = {};
      }
      current = (current as Record<string, unknown>)[k] as Record<string, unknown>;
    }

    // Set the final value
    (current as Record<string, unknown>)[keys[keys.length - 1]] = value;
  }

  return unflattened;
}

export function AppConfigDialog({
  open,
  onOpenChange,
  app,
  existingConfig,
  existingAppName,
  onSave,
  isSaving = false,
}: AppConfigDialogProps) {
  const { currentInstance } = useInstanceContext();
  const { apps: deployedApps } = useDeployedApps(currentInstance || '');
  const [appName, setAppName] = useState<string>('');
  const [nameError, setNameError] = useState<string>('');
  const [config, setConfig] = useState<Record<string, string>>({});
  const [requiredAppMappings, setRequiredAppMappings] = useState<Record<string, string>>({});

  // Check if name is unique (excluding the current app name if editing)
  const isNameUnique = (name: string): boolean => {
    if (!name || name.trim() === '') return false;
    // If editing an existing app, allow the current name
    if (existingAppName && name.toLowerCase() === existingAppName.toLowerCase()) {
      return true;
    }
    return !deployedApps.some(a => a.name.toLowerCase() === name.toLowerCase());
  };

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

      // Initialize app name
      let defaultName: string;
      if (existingAppName) {
        // If editing an existing app, use the existing name
        defaultName = existingAppName;
        setNameError('');
      } else {
        // If adding a new app, use default if unique, otherwise let user choose
        defaultName = app.name;
        if (!isNameUnique(defaultName)) {
          // Name is taken, clear it so user must provide a unique one
          defaultName = '';
          setNameError(`The name "${app.name}" is already in use. Please choose a unique name.`);
        } else {
          setNameError('');
        }
      }
      setAppName(defaultName);

      // Initialize required app mappings from app.requires
      const initialMappings: Record<string, string> = {};
      if (app.requires && app.requires.length > 0) {
        app.requires.forEach(req => {
          // Use alias as key, or name if no alias
          const key = req.alias || req.name;

          // Auto-select if there's only one matching deployed app
          const availableApps = deployedApps.filter(a => {
            const appType = a.is || a.name;
            return appType.toLowerCase() === req.name.toLowerCase() && a.status === 'deployed';
          });

          // Default to existing installedAs, or auto-select if only one option
          initialMappings[key] = req.installedAs || (availableApps.length === 1 ? availableApps[0].name : '');
        });
      }
      setRequiredAppMappings(initialMappings);
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [app, existingConfig, open, deployedApps]);

  // Validate name whenever it changes
  useEffect(() => {
    if (appName && appName.trim() !== '') {
      if (!isNameUnique(appName)) {
        setNameError(`The name "${appName}" is already in use. Please choose a unique name.`);
      } else {
        setNameError('');
      }
    } else {
      setNameError('');
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [appName, deployedApps]);

  const handleSave = () => {
    // Unflatten the config back to nested objects before saving
    const unflattenedConfig = unflattenObject(config);
    onSave(appName, unflattenedConfig, requiredAppMappings);
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

  // Check if all required dependencies are mapped
  const allDependenciesMapped = !app.requires || app.requires.every(req => {
    const key = req.alias || req.name;
    return requiredAppMappings[key] && requiredAppMappings[key].length > 0;
  });

  const canSave = allDependenciesMapped && !nameError && appName.trim() !== '';

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[80vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Configure {app.name}</DialogTitle>
          <DialogDescription>
            {app.description}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          {/* App Name Field */}
          <div className="space-y-2">
            <Label htmlFor="app-name">
              App Name
              <span className="text-red-500">*</span>
            </Label>
            <Input
              id="app-name"
              value={appName}
              onChange={(e) => setAppName(e.target.value)}
              placeholder={app.name}
              required
              disabled={!!existingAppName}
              className={nameError ? 'border-red-500' : ''}
            />
            {existingAppName ? (
              <p className="text-xs text-muted-foreground">
                App name cannot be changed after it has been added
              </p>
            ) : nameError ? (
              <p className="text-xs text-red-600">
                {nameError}
              </p>
            ) : (
              <p className="text-xs text-muted-foreground">
                Choose a name for this app installation (e.g., "postgres-primary")
              </p>
            )}
          </div>

          {/* Required Dependencies Section */}
          {app.requires && app.requires.length > 0 && (
            <div className="space-y-4">
              <div className="p-4 bg-blue-50 dark:bg-blue-950/20 rounded-lg border border-blue-200 dark:border-blue-800">
                <h4 className="text-sm font-medium text-blue-900 dark:text-blue-100 mb-2 flex items-center gap-2">
                  <Info className="h-4 w-4" />
                  Required Dependencies
                </h4>
                <p className="text-sm text-blue-800 dark:text-blue-200 mb-3">
                  This app depends on other apps. Please select which installed app should be used for each dependency:
                </p>
              </div>

              {app.requires.map(req => {
                const key = req.alias || req.name;
                const label = req.alias ? `${req.alias} (requires ${req.name})` : req.name;
                const selectedApp = requiredAppMappings[key];
                // Filter by 'is' field to find apps of the required type
                const availableApps = deployedApps.filter(a => {
                  // Check if app's 'is' field matches the required app name
                  // Fallback to name matching if 'is' is not set
                  const appType = a.is || a.name;
                  return appType.toLowerCase() === req.name.toLowerCase() &&
                         a.status === 'deployed';
                });
                const hasNoMatches = availableApps.length === 0;

                return (
                  <div key={key} className="space-y-2">
                    <Label htmlFor={`dep-${key}`}>
                      {label}
                      <span className="text-red-500">*</span>
                    </Label>
                    {hasNoMatches && (
                      <Alert variant="destructive" className="mb-2">
                        <AlertCircle className="h-4 w-4" />
                        <AlertDescription>
                          No deployed apps found matching "{req.name}". You must deploy {req.name} before adding this app.
                        </AlertDescription>
                      </Alert>
                    )}
                    <Select
                      value={selectedApp}
                      onValueChange={(value) => setRequiredAppMappings(prev => ({ ...prev, [key]: value }))}
                      disabled={hasNoMatches}
                    >
                      <SelectTrigger id={`dep-${key}`} className="mt-1">
                        <SelectValue placeholder={hasNoMatches ? "No apps available" : "Select an installed app..."} />
                      </SelectTrigger>
                      <SelectContent>
                        {availableApps.map(depApp => (
                          <SelectItem key={depApp.name} value={depApp.name}>
                            {depApp.name}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    <p className="text-xs text-muted-foreground">
                      Choose which deployed {req.name} instance to use
                    </p>
                  </div>
                );
              })}
            </div>
          )}

          {/* Configuration Fields Section */}
          {hasConfig && (
            <div className="space-y-4">
              <div className="border-t pt-4">
                <h4 className="text-sm font-medium mb-3">Configuration</h4>
              </div>
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
            </div>
          )}
        </div>


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
            disabled={isSaving || !canSave}
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
