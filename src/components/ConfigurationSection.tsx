import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { FileText, Check, AlertCircle, Loader2 } from 'lucide-react';
import { useConfig, useMessages } from '../hooks';
import { configFormSchema, defaultConfigValues, type ConfigFormData } from '../schemas/config';
import { Message } from './Message';
import { Card, CardHeader, CardTitle, CardContent, Button, Form, FormField, FormItem, FormLabel, FormControl, FormMessage, Input } from './ui';

export const ConfigurationSection = () => {
  const { 
    config, 
    isConfigured, 
    showConfigSetup, 
    isLoading, 
    isCreating, 
    error, 
    createConfig, 
    refetch 
  } = useConfig();
  const { messages } = useMessages();

  const form = useForm<ConfigFormData>({
    resolver: zodResolver(configFormSchema),
    defaultValues: defaultConfigValues,
  });

  const onSubmit = (data: ConfigFormData) => {
    createConfig(data);
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>Configuration</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <Button onClick={() => refetch()} disabled={isLoading} variant="outline">
          <FileText className="mr-2 h-4 w-4" />
          {isLoading ? 'Loading...' : 'Reload Configuration'}
        </Button>
        
        {error && (
          <div className="p-3 bg-red-50 dark:bg-red-950 border border-red-200 dark:border-red-800 rounded-md flex items-start gap-2">
            <AlertCircle className="h-4 w-4 text-red-600 mt-0.5" />
            <div>
              <p className="text-sm font-medium text-red-800 dark:text-red-200">Configuration Error</p>
              <p className="text-sm text-red-700 dark:text-red-300">{error.message}</p>
            </div>
          </div>
        )}
        
        <Message message={messages.config} />
        
        {showConfigSetup && (
          <div className="space-y-4">
            <div>
              <h3 className="text-lg font-medium">Initial Configuration Setup</h3>
              <p className="text-sm text-muted-foreground">Configure key settings for your wild-cloud central server:</p>
            </div>
            
            <Form {...form}>
              <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <FormField
                    control={form.control}
                    name="cloud.domain"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Cloud Domain</FormLabel>
                        <FormControl>
                          <Input placeholder="wildcloud.local" {...field} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={form.control}
                    name="cloud.dns.ip"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>DNS Server IP</FormLabel>
                        <FormControl>
                          <Input placeholder="192.168.8.50" {...field} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={form.control}
                    name="cloud.router.ip"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Router IP</FormLabel>
                        <FormControl>
                          <Input placeholder="192.168.8.1" {...field} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={form.control}
                    name="cloud.dnsmasq.interface"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Network Interface</FormLabel>
                        <FormControl>
                          <Input placeholder="eth0" {...field} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </div>
                
                <Button type="submit" disabled={isCreating}>
                  {isCreating ? (
                    <>
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                      Creating...
                    </>
                  ) : (
                    <>
                      <Check className="mr-2 h-4 w-4" />
                      Create Configuration
                    </>
                  )}
                </Button>
              </form>
            </Form>
          </div>
        )}
        
        {config && isConfigured && (
          <div className="space-y-2">
            <div className="p-3 bg-green-50 dark:bg-green-950 border border-green-200 dark:border-green-800 rounded-md">
              <p className="text-sm text-green-800 dark:text-green-200">
                ✓ Configuration loaded successfully
              </p>
            </div>
            <pre className="p-4 bg-muted rounded-md text-sm overflow-auto max-h-96">
              {JSON.stringify(config, null, 2)}
            </pre>
          </div>
        )}
        
        {/* Debug info */}
        <div className="text-xs text-muted-foreground">
          React Query Status: isLoading={isLoading.toString()}, isConfigured={isConfigured.toString()}, showSetup={showConfigSetup.toString()}
        </div>
      </CardContent>
    </Card>
  );
};