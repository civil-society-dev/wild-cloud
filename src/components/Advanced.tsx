import { useState } from "react";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "./ui/card";
import { ConfigEditor } from "./ConfigEditor";
import { Button, Input, Label } from "./ui";
import { Check, Edit2, HelpCircle, X } from "lucide-react";

export function Advanced() {
  const [upstreamValue, setUpstreamValue] = useState("https://mywildcloud.org");
  const [editingUpstream, setEditingUpstream] = useState(false);
  const [tempUpstream, setTempUpstream] = useState(upstreamValue);
  const handleUpstreamEdit = () => {
    setTempUpstream(upstreamValue);
    setEditingUpstream(true);
  };

  const handleUpstreamSave = () => {
    setUpstreamValue(tempUpstream);
    setEditingUpstream(false);
  };

  const handleUpstreamCancel = () => {
    setTempUpstream(upstreamValue);
    setEditingUpstream(false);
  };

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>Advanced Configuration</CardTitle>
          <CardDescription>
            Advanced settings and system configuration options
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div>
            <h3 className="text-sm font-medium mb-2">
              Configuration Management
            </h3>
            <p className="text-sm text-muted-foreground mb-4">
              Edit the raw YAML configuration file directly. This provides full
              access to all configuration options.
            </p>
            <ConfigEditor />
          </div>
        </CardContent>
      </Card>
      {/* Upstream Section */}
      <Card className="p-4 border-l-4 border-l-blue-500">
        <div className="flex items-center justify-between mb-3">
          <div>
            <h3 className="font-medium">Upstream Configuration</h3>
            <p className="text-sm text-muted-foreground">
              External service endpoint
            </p>
          </div>
          <div className="flex items-center gap-2">
            <Button variant="ghost" size="sm">
              <HelpCircle className="h-4 w-4" />
            </Button>
            {!editingUpstream && (
              <Button variant="outline" size="sm" onClick={handleUpstreamEdit}>
                <Edit2 className="h-4 w-4 mr-1" />
                Edit
              </Button>
            )}
          </div>
        </div>

        {editingUpstream ? (
          <div className="space-y-3">
            <div>
              <Label htmlFor="upstream-edit">Upstream URL</Label>
              <Input
                id="upstream-edit"
                value={tempUpstream}
                onChange={(e) => setTempUpstream(e.target.value)}
                placeholder="https://example.com"
                className="mt-1"
              />
            </div>
            <div className="flex gap-2">
              <Button size="sm" onClick={handleUpstreamSave}>
                <Check className="h-4 w-4 mr-1" />
                Save
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={handleUpstreamCancel}
              >
                <X className="h-4 w-4 mr-1" />
                Cancel
              </Button>
            </div>
          </div>
        ) : (
          <div>
            <Label>Upstream URL</Label>
            <div className="mt-1 p-2 bg-muted rounded-md font-mono text-sm">
              {upstreamValue}
            </div>
          </div>
        )}
      </Card>
    </div>
  );
}
