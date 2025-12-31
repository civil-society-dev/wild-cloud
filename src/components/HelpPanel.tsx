import { useHelp } from '../contexts/HelpContext';
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription } from './ui/sheet';
import { Card } from './ui/card';

export function HelpPanel() {
  const { helpContent, isHelpOpen, setIsHelpOpen } = useHelp();

  if (!helpContent) return null;

  return (
    <Sheet open={isHelpOpen} onOpenChange={setIsHelpOpen}>
      <SheetContent className="w-full sm:max-w-lg overflow-y-auto p-6">
        <SheetHeader className="mb-6">
          <SheetTitle className="flex items-center gap-3">
            {helpContent.icon && (
              <div className={`p-2 rounded-lg ${helpContent.color || 'bg-primary/10'}`}>
                {helpContent.icon}
              </div>
            )}
            {helpContent.title}
          </SheetTitle>
          <SheetDescription className="sr-only">
            Help information for {helpContent.title}
          </SheetDescription>
        </SheetHeader>
        <div className="space-y-4">
          <Card className={`p-4 ${helpContent.color || 'bg-muted/50'}`}>
            <div className="prose dark:prose-invert max-w-none text-sm">
              {helpContent.description}
            </div>
          </Card>
          {helpContent.actions && (
            <div className="flex gap-2 flex-wrap">
              {helpContent.actions}
            </div>
          )}
        </div>
      </SheetContent>
    </Sheet>
  );
}
