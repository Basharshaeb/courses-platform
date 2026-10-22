import Link from "next/link";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

/** Centered card used by every login/signup page. */
export function AuthCard({
  title,
  children,
  footer,
}: {
  title: string;
  children: React.ReactNode;
  footer?: { text: string; href: string; link: string };
}) {
  return (
    <main className="flex flex-1 items-center justify-center p-4">
      <Card className="w-full max-w-sm">
        <CardHeader>
          <CardTitle className="text-xl">{title}</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-4">
          {children}
          {footer && (
            <p className="text-center text-sm text-muted-foreground">
              {footer.text}{" "}
              <Link href={footer.href} className="font-medium text-foreground underline underline-offset-4">
                {footer.link}
              </Link>
            </p>
          )}
        </CardContent>
      </Card>
    </main>
  );
}
