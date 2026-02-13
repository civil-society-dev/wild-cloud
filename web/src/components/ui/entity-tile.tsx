import * as React from "react"

import { cn } from "@/lib/utils"
import { Badge } from "@/components/ui/badge"

interface EntityTileProps extends React.ComponentProps<"div"> {
  icon?: React.ReactNode
  title: string
  version?: string
  description?: string
  statusIndicator?: React.ReactNode
  onClick?: () => void
  disabled?: boolean
  tint?: string
}

function EntityTile({
  icon,
  title,
  version,
  description,
  statusIndicator,
  onClick,
  disabled = false,
  tint,
  className,
  children,
  ...props
}: EntityTileProps) {
  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (disabled || !onClick) return
    if (e.key === "Enter" || e.key === " ") {
      e.preventDefault()
      onClick()
    }
  }

  return (
    <div className="flex flex-col items-center">
      <div
        role={onClick ? "button" : undefined}
        tabIndex={onClick && !disabled ? 0 : undefined}
        onClick={!disabled ? onClick : undefined}
        onKeyDown={onClick ? handleKeyDown : undefined}
        aria-disabled={disabled || undefined}
        className={cn(
          "w-full aspect-square text-black border border-border/60 p-4 flex flex-col rounded-none",
          "transition-all duration-200",
          onClick && !disabled && [
            "cursor-pointer",
            "hover:shadow-md hover:border-primary/50",
            "hover:scale-[1.02] hover:-translate-y-0.5",
            "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
          ],
          disabled && "opacity-50 pointer-events-none",
          className
        )}
        {...props}
        style={tint ? { backgroundColor: `${tint}FF` } : undefined}
      >
        <div className="flex flex-col gap-3 flex-1">
          <div className="flex items-start gap-3">
            {icon && (
              <div className="h-10 w-10 rounded-lg bg-white/70 flex items-center justify-center overflow-hidden shrink-0">
                {icon}
              </div>
            )}
            <div className="flex-1 min-w-0">
              <h4 className="font-medium truncate mb-1">{title}</h4>
              {version && (
                <Badge variant="outline" className="text-xs text-black/70 border-black/20">
                  {version}
                </Badge>
              )}
            </div>
          </div>
          {description && (
            <p className="text-sm text-black/60 line-clamp-2">{description}</p>
          )}
        </div>
        {children}
      </div>
      <div className="flex justify-center mt-2 h-3">
        {statusIndicator}
      </div>
    </div>
  )
}

export { EntityTile }
export type { EntityTileProps }
