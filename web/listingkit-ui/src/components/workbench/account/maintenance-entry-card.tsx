import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import overviewStyles from "../console/primary-overview.module.css";

export function MaintenanceEntryCard({ title, description, action, href }: {
  title: string;
  description: string;
  action: string;
  href: string;
}) {
  return <Card className={overviewStyles.entry}>
    <h2>{title}</h2>
    <p className={overviewStyles.description}>{description}</p>
    <Button asChild size="sm" variant="outline"><Link href={href} prefetch={false}>{action}</Link></Button>
  </Card>;
}
