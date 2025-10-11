import { useState } from "react";
import { Card } from "./ui/card";
import { Button } from "./ui/button";
import { Cloud, HelpCircle, Edit2, Check, X } from "lucide-react";
import { Input, Label } from "./ui";

export function CloudComponent() {
  const [domainValue, setDomainValue] = useState("cloud.payne.io");
  const [internalDomainValue, setInternalDomainValue] = useState(
    "internal.cloud.payne.io"
  );

  const [editingDomains, setEditingDomains] = useState(false);

  const [tempDomain, setTempDomain] = useState(domainValue);
  const [tempInternalDomain, setTempInternalDomain] =
    useState(internalDomainValue);

  const handleDomainsEdit = () => {
    setTempDomain(domainValue);
    setTempInternalDomain(internalDomainValue);
    setEditingDomains(true);
  };

  const handleDomainsSave = () => {
    setDomainValue(tempDomain);
    setInternalDomainValue(tempInternalDomain);
    setEditingDomains(false);
  };

  const handleDomainsCancel = () => {
    setTempDomain(domainValue);
    setTempInternalDomain(internalDomainValue);
    setEditingDomains(false);
  };

  return (
    <div className="space-y-6">
      <Card className="p-6">
        <div className="flex items-center gap-4 mb-6">
          <div className="p-2 bg-primary/10 rounded-lg">
            <Cloud className="h-6 w-6 text-primary" />
          </div>
          <div>
            <h2 className="text-2xl font-semibold">Cloud Configuration</h2>
            <p className="text-muted-foreground">
              Configure top-level cloud settings and domains
            </p>
          </div>
        </div>

        <div className="space-y-6">
          {/* Domains Section */}
          <Card className="p-4 border-l-4 border-l-green-500">
            <div className="flex items-center justify-between mb-3">
              <div>
                <h3 className="font-medium">Domain Configuration</h3>
                <p className="text-sm text-muted-foreground">
                  Public and internal domain settings
                </p>
              </div>
              <div className="flex items-center gap-2">
                <Button variant="ghost" size="sm">
                  <HelpCircle className="h-4 w-4" />
                </Button>
                {!editingDomains && (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleDomainsEdit}
                  >
                    <Edit2 className="h-4 w-4 mr-1" />
                    Edit
                  </Button>
                )}
              </div>
            </div>

            {editingDomains ? (
              <div className="space-y-3">
                <div>
                  <Label htmlFor="domain-edit">Public Domain</Label>
                  <Input
                    id="domain-edit"
                    value={tempDomain}
                    onChange={(e) => setTempDomain(e.target.value)}
                    placeholder="example.com"
                    className="mt-1"
                  />
                </div>
                <div>
                  <Label htmlFor="internal-domain-edit">Internal Domain</Label>
                  <Input
                    id="internal-domain-edit"
                    value={tempInternalDomain}
                    onChange={(e) => setTempInternalDomain(e.target.value)}
                    placeholder="internal.example.com"
                    className="mt-1"
                  />
                </div>
                <div className="flex gap-2">
                  <Button size="sm" onClick={handleDomainsSave}>
                    <Check className="h-4 w-4 mr-1" />
                    Save
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleDomainsCancel}
                  >
                    <X className="h-4 w-4 mr-1" />
                    Cancel
                  </Button>
                </div>
              </div>
            ) : (
              <div className="space-y-3">
                <div>
                  <Label>Public Domain</Label>
                  <div className="mt-1 p-2 bg-muted rounded-md font-mono text-sm">
                    {domainValue}
                  </div>
                </div>
                <div>
                  <Label>Internal Domain</Label>
                  <div className="mt-1 p-2 bg-muted rounded-md font-mono text-sm">
                    {internalDomainValue}
                  </div>
                </div>
              </div>
            )}
          </Card>
        </div>
      </Card>
    </div>
  );
}
