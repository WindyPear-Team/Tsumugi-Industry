import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog"

export function ConfirmDialog({ open, title, description, onOpenChange, onConfirm, confirmLabel = "确认", destructive = true }: { open: boolean; title: string; description: string; onOpenChange: (open: boolean) => void; onConfirm: () => void; confirmLabel?: string; destructive?: boolean }) {
  return <AlertDialog open={open} onOpenChange={onOpenChange}><AlertDialogContent><AlertDialogHeader><AlertDialogTitle>{title}</AlertDialogTitle><AlertDialogDescription>{description}</AlertDialogDescription></AlertDialogHeader><AlertDialogFooter><AlertDialogCancel>取消</AlertDialogCancel><AlertDialogAction className={destructive ? "bg-destructive text-white hover:bg-destructive/90" : undefined} onClick={onConfirm}>{confirmLabel}</AlertDialogAction></AlertDialogFooter></AlertDialogContent></AlertDialog>
}
