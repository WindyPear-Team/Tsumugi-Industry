import { ArrowDown, ArrowUp, ArrowUpDown, EyeOff, Filter, X } from "lucide-react"
import { Button } from "@/components/ui/button"
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuLabel, DropdownMenuSeparator, DropdownMenuTrigger } from "@/components/ui/dropdown-menu"
import { Input } from "@/components/ui/input"

type Props = { label: string; field: string; filter?: string; sortField?: string; sortOrder?: "asc" | "desc"; onFilter: (value: string) => void; onSort: (field: string, order: "asc" | "desc") => void }

export function TableColumnMenu({ label, field, filter = "", sortField, sortOrder, onFilter, onSort }: Props) {
  const active = filter !== "" || sortField === field
  return <DropdownMenu><DropdownMenuTrigger asChild><Button variant="ghost" size="sm" className={active ? "bg-muted" : "-ml-3"}>{label}{sortField === field ? sortOrder === "asc" ? <ArrowUp /> : <ArrowDown /> : <ArrowUpDown />}{filter && <Filter className="text-primary" />}</Button></DropdownMenuTrigger><DropdownMenuContent align="start" className="w-64"><DropdownMenuLabel>排序与筛选</DropdownMenuLabel><DropdownMenuItem onSelect={() => onSort(field, "asc")}><ArrowUp />升序</DropdownMenuItem><DropdownMenuItem onSelect={() => onSort(field, "desc")}><ArrowDown />降序</DropdownMenuItem><DropdownMenuSeparator /><div className="px-2 py-1.5" onKeyDown={(event) => event.stopPropagation()}><Input autoFocus={false} placeholder={`筛选${label}`} value={filter} onChange={(event) => onFilter(event.target.value)} /></div><DropdownMenuItem onSelect={() => onFilter("")}><X />清除筛选</DropdownMenuItem><DropdownMenuSeparator /><DropdownMenuItem disabled><EyeOff />隐藏列（预留）</DropdownMenuItem></DropdownMenuContent></DropdownMenu>
}
