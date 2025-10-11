import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { FileText, Check, AlertCircle, Loader2 } from 'lucide-react';
import { useConfig, useMessages } from '../hooks';
import { configFormSchema, defaultConfigValues, type ConfigFormData } from '../schemas/config';
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  Button,
  Form,
  FormField,
  FormItem,
  FormLabel,
  FormControl,
  FormDescription,
  FormMessage,
  Input,
} from './ui';

export const ConfigurationForm = () => {
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
        <CardTitle>Configuration (With Form Validation)</CardTitle>
      </CardHeader>
      <CardContent className="space-y-6">
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

        {showConfigSetup && (
          <div className="space-y-6">
            <div>
              <h3 className="text-lg font-medium">Initial Configuration Setup</h3>
              <p className="text-sm text-muted-foreground">Configure your wild-cloud central server settings with real-time validation.</p>
            </div>
            
            <Form {...form}>
              <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
                {/* Server Configuration */}
                <div className="space-y-4">
                  <h4 className="text-md font-medium text-foreground">Server Configuration</h4>
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <FormField
                      control={form.control}
                      name="server.host"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>Server Host</FormLabel>
                          <FormControl>
                            <Input placeholder="0.0.0.0" {...field} />
                          </FormControl>
                          <FormDescription>
                            The host address the server will bind to
                          </FormDescription>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                    <FormField
                      control={form.control}
                      name="server.port"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>Server Port</FormLabel>
                          <FormControl>
                            <Input 
                              type="number" 
                              placeholder="5055" 
                              {...field}
                              onChange={(e) => field.onChange(parseInt(e.target.value) || 0)}
                            />
                          </FormControl>
                          <FormDescription>
                            The port the server will listen on
                          </FormDescription>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                  </div>
                </div>

                {/* Cloud Configuration */}
                <div className="space-y-4">
                  <h4 className="text-md font-medium text-foreground">Cloud Configuration</h4>
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <FormField
                      control={form.control}
                      name="cloud.domain"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>Domain</FormLabel>
                          <FormControl>
                            <Input placeholder="wildcloud.local" {...field} />
                          </FormControl>
                          <FormDescription>
                            The main domain for your wild-cloud
                          </FormDescription>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                    <FormField
                      control={form.control}
                      name="cloud.internalDomain"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>Internal Domain</FormLabel>
                          <FormControl>
                            <Input placeholder="cluster.local" {...field} />
                          </FormControl>
                          <FormDescription>
                            The internal cluster domain
                          </FormDescription>
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
                          <FormDescription>
                            The IP address of the DNS server
                          </FormDescription>
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
                          <FormDescription>
                            The IP address of the network router
                          </FormDescription>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                    <FormField
                      control={form.control}
                      name="cloud.dhcpRange"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>DHCP Range</FormLabel>
                          <FormControl>
                            <Input placeholder="192.168.8.100,192.168.8.200" {...field} />
                          </FormControl>
                          <FormDescription>
                            DHCP IP range in format: start_ip,end_ip
                          </FormDescription>
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
                          <FormDescription>
                            The network interface for dnsmasq to use
                          </FormDescription>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                  </div>
                </div>

                {/* Cluster Configuration */}
                <div className="space-y-4">
                  <h4 className="text-md font-medium text-foreground">Cluster Configuration</h4>
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <FormField
                      control={form.control}
                      name="cluster.endpointIp"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>Cluster Endpoint IP</FormLabel>
                          <FormControl>
                            <Input placeholder="192.168.8.60" {...field} />
                          </FormControl>
                          <FormDescription>
                            The IP address of the cluster endpoint
                          </FormDescription>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                    <FormField
                      control={form.control}
                      name="cluster.nodes.talos.version"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>Talos Version</FormLabel>
                          <FormControl>
                            <Input placeholder="v1.8.0" {...field} />
                          </FormControl>
                          <FormDescription>
                            The version of Talos Linux to use
                          </FormDescription>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                  </div>
                </div>

                <Button type="submit" disabled={isCreating} className="w-full">
                  {isCreating ? (
                    <>
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                      Creating Configuration...
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
        
        <div className="text-xs text-muted-foreground">
          Form Validation Status: {form.formState.isValid ? '✓ Valid' : '⚠ Has Errors'} | 
          Errors: {Object.keys(form.formState.errors).length}
        </div>
      </CardContent>
    </Card>
  );
};