"use client";

import { use, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button, buttonVariants } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

export default function CheckoutPage({
  params,
}: {
  params: Promise<{ host: string; slug: string }>;
}) {
  const { slug } = use(params);
  const router = useRouter();

  const isGoCourse = slug.includes("go");
  const courseTitle = isGoCourse
    ? "بناء الأنظمة الموزعة والخدمات المصغرة بلغة Go"
    : "تطوير تطبيقات الويب المتكاملة مع Next.js 16 و React";

  const priceAmount = isGoCourse ? 349 : 289;
  const vatAmount = Math.round(priceAmount * 0.15);
  const totalAmount = priceAmount; // inclusive of VAT

  const [paymentMethod, setPaymentMethod] = useState<"mada" | "apple_pay" | "card">("mada");
  const [fullName, setFullName] = useState("");
  const [email, setEmail] = useState("");
  const [cardNumber, setCardNumber] = useState("");
  const [expiry, setExpiry] = useState("");
  const [cvc, setCvc] = useState("");
  const [isProcessing, setIsProcessing] = useState(false);
  const [isSuccess, setIsSuccess] = useState(false);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setIsProcessing(true);

    // Simulate gateway authorization with Moyasar/mada
    setTimeout(() => {
      setIsProcessing(false);
      setIsSuccess(true);
      setTimeout(() => {
        router.push("/library");
      }, 2000);
    }, 1200);
  };

  return (
    <div className="min-h-screen bg-muted/20 py-10 px-4 sm:px-6 text-foreground selection:bg-primary/20 selection:text-primary">
      <div className="mx-auto max-w-4xl space-y-8">
        {/* Navigation Breadcrumb */}
        <nav className="flex items-center gap-2 text-xs text-muted-foreground font-medium">
          <Link href="/" className="hover:text-foreground transition-colors">
            الرئيسية
          </Link>
          <span>/</span>
          <Link href={`/courses/${slug}`} className="hover:text-foreground transition-colors">
            {courseTitle}
          </Link>
          <span>/</span>
          <span className="text-foreground">إتمام الطلب والدفع</span>
        </nav>

        {isSuccess ? (
          <Card className="border-emerald-500/40 bg-card shadow-lg text-center p-8 sm:p-12 animate-in fade-in zoom-in-95 duration-300">
            <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-emerald-500/15 text-emerald-500 text-3xl font-bold">
              ✓
            </div>
            <h2 className="text-2xl sm:text-3xl font-extrabold text-foreground mb-2">
              تم الدفع وتأكيد تسجيلك بنجاح!
            </h2>
            <p className="text-sm text-muted-foreground max-w-md mx-auto mb-6">
              تم إصدار فاتورة ضريبية مبسطة وإرسال تفاصيل الدخول إلى بريدك الإلكتروني. جاري تحويلك إلى مكتبتك التعليمية...
            </p>
            <div className="flex justify-center gap-3">
              <Link
                href="/library"
                className={buttonVariants({
                  className: "shadow-md font-semibold",
                })}
              >
                الانتقال إلى مكتبة دوراتي
              </Link>
            </div>
          </Card>
        ) : (
          <div className="grid gap-8 lg:grid-cols-12 items-start">
            {/* Left Column: Payment Form (8 cols on lg) */}
            <div className="lg:col-span-7 space-y-6">
              {/* Step 1: Customer Account Info */}
              <Card className="border-border/80 shadow-xs">
                <CardHeader className="pb-4">
                  <div className="flex items-center gap-2">
                    <span className="flex h-6 w-6 items-center justify-center rounded-full bg-primary text-primary-foreground text-xs font-bold">
                      1
                    </span>
                    <CardTitle className="text-base font-bold">معلومات الطالب للتسجيل</CardTitle>
                  </div>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="space-y-2">
                    <Label htmlFor="fullname" className="text-xs font-semibold">
                      الاسم الكامل (سيظهر على شهادة الإتمام)
                    </Label>
                    <Input
                      id="fullname"
                      placeholder="محمد عبدالله"
                      value={fullName}
                      onChange={(e) => setFullName(e.target.value)}
                      required
                      className="bg-background"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="email" className="text-xs font-semibold">
                      البريد الإلكتروني للوصول إلى الدروس
                    </Label>
                    <Input
                      id="email"
                      type="email"
                      placeholder="student@example.com"
                      value={email}
                      onChange={(e) => setEmail(e.target.value)}
                      required
                      className="bg-background text-start"
                      dir="ltr"
                    />
                  </div>
                </CardContent>
              </Card>

              {/* Step 2: Payment Method */}
              <Card className="border-border/80 shadow-xs">
                <CardHeader className="pb-4">
                  <div className="flex items-center gap-2">
                    <span className="flex h-6 w-6 items-center justify-center rounded-full bg-primary text-primary-foreground text-xs font-bold">
                      2
                    </span>
                    <CardTitle className="text-base font-bold">طريقة الدفع الآمن</CardTitle>
                  </div>
                </CardHeader>
                <CardContent className="space-y-5">
                  {/* Payment selector tabs */}
                  <div className="grid grid-cols-3 gap-2.5">
                    <button
                      type="button"
                      onClick={() => setPaymentMethod("mada")}
                      className={`flex flex-col items-center justify-center gap-1.5 rounded-xl border p-3 text-xs font-semibold transition-all ${
                        paymentMethod === "mada"
                          ? "border-primary bg-primary/10 text-primary shadow-xs ring-1 ring-primary"
                          : "border-border/80 bg-background hover:bg-muted/40 text-muted-foreground"
                      }`}
                    >
                      <span className="text-base">💳</span>
                      <span>بطاقة مدى</span>
                    </button>

                    <button
                      type="button"
                      onClick={() => setPaymentMethod("apple_pay")}
                      className={`flex flex-col items-center justify-center gap-1.5 rounded-xl border p-3 text-xs font-semibold transition-all ${
                        paymentMethod === "apple_pay"
                          ? "border-primary bg-primary/10 text-primary shadow-xs ring-1 ring-primary"
                          : "border-border/80 bg-background hover:bg-muted/40 text-muted-foreground"
                      }`}
                    >
                      <span className="text-base"></span>
                      <span>Apple Pay</span>
                    </button>

                    <button
                      type="button"
                      onClick={() => setPaymentMethod("card")}
                      className={`flex flex-col items-center justify-center gap-1.5 rounded-xl border p-3 text-xs font-semibold transition-all ${
                        paymentMethod === "card"
                          ? "border-primary bg-primary/10 text-primary shadow-xs ring-1 ring-primary"
                          : "border-border/80 bg-background hover:bg-muted/40 text-muted-foreground"
                      }`}
                    >
                      <span className="text-base">🌐</span>
                      <span>فيزا / ماستركارد</span>
                    </button>
                  </div>

                  {paymentMethod === "apple_pay" ? (
                    <div className="rounded-xl border border-dashed border-border/80 bg-muted/20 p-6 text-center space-y-3">
                      <p className="text-xs text-muted-foreground">
                        اضغط على الزر أدناه لتأكيد الدفع فوراً عبر Apple Pay وبصمة الوجه Face ID
                      </p>
                      <Button
                        type="button"
                        onClick={handleSubmit}
                        disabled={isProcessing}
                        className="w-full bg-black text-white hover:bg-black/90 h-11 text-base font-semibold tracking-tight shadow-md"
                      >
                        {isProcessing ? "جاري المعالجة..." : "Pay الدفع السريع"}
                      </Button>
                    </div>
                  ) : (
                    <form onSubmit={handleSubmit} className="space-y-4">
                      <div className="space-y-2">
                        <Label htmlFor="cardnumber" className="text-xs font-semibold">
                          رقم البطاقة (مدى أو الائتمان)
                        </Label>
                        <Input
                          id="cardnumber"
                          placeholder="4000 1234 5678 9010"
                          value={cardNumber}
                          onChange={(e) => setCardNumber(e.target.value)}
                          required
                          dir="ltr"
                          className="bg-background text-start font-mono"
                        />
                      </div>

                      <div className="grid grid-cols-2 gap-3">
                        <div className="space-y-2">
                          <Label htmlFor="expiry" className="text-xs font-semibold">
                            تاريخ الانتهاء
                          </Label>
                          <Input
                            id="expiry"
                            placeholder="MM / YY"
                            value={expiry}
                            onChange={(e) => setExpiry(e.target.value)}
                            required
                            dir="ltr"
                            className="bg-background text-start font-mono"
                          />
                        </div>
                        <div className="space-y-2">
                          <Label htmlFor="cvc" className="text-xs font-semibold">
                            رمز الأمان (CVC)
                          </Label>
                          <Input
                            id="cvc"
                            placeholder="123"
                            value={cvc}
                            onChange={(e) => setCvc(e.target.value)}
                            required
                            dir="ltr"
                            maxLength={4}
                            className="bg-background text-start font-mono"
                          />
                        </div>
                      </div>

                      <Button
                        type="submit"
                        disabled={isProcessing}
                        className="w-full h-11 text-base font-semibold shadow-md gap-2"
                      >
                        {isProcessing ? (
                          <span>جاري التحقق وإتمام الدفع...</span>
                        ) : (
                          <>
                            <span>إتمام التسجيل والدفع الآن ({totalAmount} ر.س)</span>
                            <span>←</span>
                          </>
                        )}
                      </Button>
                    </form>
                  )}

                  <div className="flex items-center justify-center gap-4 pt-2 text-[11px] text-muted-foreground">
                    <span className="flex items-center gap-1">🔒 تشفير بنكي 256-bit</span>
                    <span>•</span>
                    <span>معتمد من Moyasar & SAMA</span>
                    <span>•</span>
                    <span>ضمان استرداد 14 يوم</span>
                  </div>
                </CardContent>
              </Card>
            </div>

            {/* Right Column: Order Summary (5 cols on lg) */}
            <div className="lg:col-span-5 space-y-5">
              <Card className="border-border/80 bg-card shadow-xs">
                <CardHeader className="pb-3 border-b">
                  <CardTitle className="text-base font-bold">ملخص الطلب</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4 pt-4">
                  <div className="space-y-1">
                    <span className="text-[10px] uppercase font-bold text-primary tracking-wider">
                      الدورة التدريبية
                    </span>
                    <h3 className="font-bold text-sm leading-snug">{courseTitle}</h3>
                    <p className="text-xs text-muted-foreground">
                      وصول كامل وشامل لجميع الدروس والمشاريع والتحديثات مدى الحياة
                    </p>
                  </div>

                  <div className="space-y-2 pt-3 border-t text-xs">
                    <div className="flex justify-between text-muted-foreground">
                      <span>سعر الدورة:</span>
                      <span className="font-medium text-foreground">{priceAmount - vatAmount} ر.س</span>
                    </div>
                    <div className="flex justify-between text-muted-foreground">
                      <span>ضريبة القيمة المضافة (15% ZATCA):</span>
                      <span className="font-medium text-foreground">{vatAmount} ر.س</span>
                    </div>
                    <div className="flex justify-between text-muted-foreground text-[11px]">
                      <span>رسوم المعالجة البنكية:</span>
                      <span className="text-emerald-600 dark:text-emerald-400 font-medium">مجاناً 100%</span>
                    </div>

                    <div className="flex justify-between items-baseline pt-3 border-t text-base font-extrabold text-foreground">
                      <span>الإجمالي المطلوب:</span>
                      <div className="text-end">
                        <span className="text-2xl text-primary">{totalAmount}</span>
                        <span className="text-xs font-normal text-muted-foreground ms-1">ر.س</span>
                      </div>
                    </div>
                  </div>

                  <div className="rounded-xl bg-primary/5 border border-primary/15 p-3.5 space-y-2 text-xs">
                    <p className="font-bold text-foreground">مميزات الاشتراك الفوري:</p>
                    <ul className="space-y-1.5 text-muted-foreground text-[11px]">
                      <li className="flex items-center gap-1.5">
                        <span className="text-primary font-bold">✓</span>
                        تفعيل فوري للحساب والدروس بعد الدفع مباشرة
                      </li>
                      <li className="flex items-center gap-1.5">
                        <span className="text-primary font-bold">✓</span>
                        شهادة إتمام رقمية معتمدة برمز تحقق فوري
                      </li>
                      <li className="flex items-center gap-1.5">
                        <span className="text-primary font-bold">✓</span>
                        فاتورة ضريبية رسمية متوافقة مع هيئة الزكاة والضريبة
                      </li>
                    </ul>
                  </div>
                </CardContent>
              </Card>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
