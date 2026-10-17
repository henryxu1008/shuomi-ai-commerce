"use client"

import * as React from "react"
import { cn } from "@/vendor/tiptap/lib/tiptap-utils"
import "@/vendor/tiptap/components/tiptap-ui-primitive/card/card.css"

const Card = React.forwardRef<HTMLDivElement, React.ComponentProps<"div">>(
  ({ className, ...props }, ref) => {
    return <div ref={ref} className={cn("tiptap-card", className)} {...props} />
  }
)
Card.displayName = "Card"

const CardBody = React.forwardRef<HTMLDivElement, React.ComponentProps<"div">>(
  ({ className, ...props }, ref) => {
    return (
      <div ref={ref} className={cn("tiptap-card-body", className)} {...props} />
    )
  }
)
CardBody.displayName = "CardBody"

const CardItemGroup = React.forwardRef<
  HTMLDivElement,
  React.ComponentProps<"div"> & {
    orientation?: "horizontal" | "vertical"
  }
>(({ className, orientation = "vertical", ...props }, ref) => {
  return (
    <div
      ref={ref}
      data-orientation={orientation}
      className={cn("tiptap-card-item-group", className)}
      {...props}
    />
  )
})
CardItemGroup.displayName = "CardItemGroup"

export { Card, CardBody, CardItemGroup }
