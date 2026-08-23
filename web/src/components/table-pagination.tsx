import { ChevronLeft, ChevronRight } from "lucide-react"
import { Button } from "@/components/ui/button"

export function TablePagination({ page, onPageChange }: { page: { page: number; page_count: number; total: number; page_size: number }; onPageChange: (page: number) => void }) {
  return <div className="flex items-center justify-between border-t border-border/70 pt-4 text-xs text-muted-foreground"><span>共 {page.total} 条，第 {page.page} / {Math.max(page.page_count, 1)} 页</span><div className="flex items-center gap-1"><Button variant="outline" size="icon-sm" disabled={page.page <= 1} onClick={() => onPageChange(page.page - 1)}><ChevronLeft /></Button><Button variant="outline" size="icon-sm" disabled={page.page >= page.page_count} onClick={() => onPageChange(page.page + 1)}><ChevronRight /></Button></div></div>
}
