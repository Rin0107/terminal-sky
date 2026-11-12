; terminal-sky の描画の仕様 (internal/sky/render.go)。
; 決まり: 証明したい性質の「否定」を assert し、(check-sat) が unsat なら証明できた。
; 実装を確かめるテスト: TestSpec_Layout・TestSpec_SymbolMonotonic・TestSpec_ProjectInside・TestSpec_SegmentShown・
; TestSpec_LineInsideHorizon・TestSpec_LineGlyph (internal/sky/spec_test.go)、
; TestSpec_MoonAge・TestSpec_BodyFits (internal/sky/planet_spec_test.go)、
; TestSpec_Compass16・TestSpec_HighlightLines (internal/sky/tonight_spec_test.go)、
; TestSpec_FacingInside・TestSpec_RelAz (internal/sky/facing_test.go)、TestSpec_ExitCode (internal/app/exit_spec_test.go)。

; 1. layout: 図の高さ h (6〜199。--height 8〜200 から見出しと凡例の 2 行を除いた高さを含む) と幅 w (20〜400) のどれでも、
;    地平線の楕円 (中心 cx, cy・半径 rx, ry) が格子に収まり、半径は 1 以上。
;    Go の整数の割り算は、0 以上の数どうしでは div と同じ。
(push)
(declare-const w Int)
(declare-const h Int)
(assert (and (<= 20 w) (<= w 400) (<= 6 h) (<= h 199)))
(define-fun ry0 () Int (div (- h 1) 2))
(define-fun lim () Int (div (- w 1) 2))
(define-fun rx () Int (ite (> (* 2 ry0) lim) lim (* 2 ry0)))
(define-fun ry () Int (ite (> (* 2 ry0) lim) (div lim 2) ry0))
(define-fun cx () Int (div (- w 1) 2))
(define-fun cy () Int ry)
(assert (not (and (>= ry 1) (>= rx 1)
                  (<= 0 (- cx rx)) (<= (+ cx rx) (- w 1))
                  (<= 0 (- cy ry)) (<= (+ cy ry) (- h 1)))))
(check-sat)
(pop)

; 2. Symbol: 等級が小さい (明るい) ほど、目立つ文字 (● = 3 > * = 2 > + = 1) になる。
(push)
(define-fun rank ((m Real)) Int (ite (< m 0.5) 3 (ite (< m 1.5) 2 1)))
(declare-const a Real)
(declare-const b Real)
(assert (<= a b))
(assert (not (>= (rank a) (rank b))))
(check-sat)
(pop)

; 3. project: 高度 0〜90 の星は r = (90 - alt) / 90 が 0〜1 で、|r・cos|・|r・sin| <= 1。
;    だから行のずれ x = -cos・r・ry は -ry〜ry の実数。それを四捨五入した整数 k (|k - x| <= 1/2) も -ry〜ry。
;    (列も同じで、ry を rx に読み替える)
(push)
(declare-const ry Int)
(declare-const x Real)
(declare-const k Int)
(assert (>= ry 1))
(assert (and (<= (- (to_real ry)) x) (<= x (to_real ry))))
(assert (and (<= (- (to_real k) x) 0.5) (<= (- x (to_real k)) 0.5)))
(assert (not (and (<= (- ry) k) (<= k ry))))
(check-sat)
(pop)

; ---- 星座線 (internal/sky/constellation.go・render.go)。実装を確かめるテスト:
;      TestSpec_SegmentShown・TestSpec_LineInsideHorizon・TestSpec_LineGlyph (internal/sky/spec_test.go)。

; 4. segmentShown: 線を描くのは、両端の少なくとも一方が地平線より上 (高度 > 0) のときだけ。
;    両端とも地平線より下 (高度 <= 0) なら描かない。
(push)
(define-fun shown ((a Real) (b Real)) Bool (or (> a 0.0) (> b 0.0)))
(declare-const a Real)
(declare-const b Real)
(assert (not (= (shown a b) (not (and (<= a 0.0) (<= b 0.0))))))
(check-sat)
(pop)

; 5. inside: 線の点は地平線の楕円の内側 (dr^2・rx^2 + dc^2・ry^2 < rx^2・ry^2) にだけ置く。
;    その点は |dr| <= ry・|dc| <= rx なので、1 の楕円と同じく格子に収まる。
(push)
(declare-const rx Int)
(declare-const ry Int)
(declare-const dr Int)
(declare-const dc Int)
(assert (and (>= rx 1) (>= ry 1)))
(assert (< (+ (* dr dr rx rx) (* dc dc ry ry)) (* rx rx ry ry)))
(assert (not (and (<= (- ry) dr) (<= dr ry) (<= (- rx) dc) (<= dc rx))))
(check-sat)
(pop)

; 6. lineGlyph: 線の文字は線の向き (行のずれ dr・列のずれ dc。行は下が正) で決まり、見た目の傾きと合う。
;    文字は縦長 (1 行 ≒ 2 列) なので、見た目の縦は 2|dr|。
;    '-' はほぼ水平 (5・2|dr| <= 2|dc|)、'|' はほぼ垂直、'/' は右上がり (dr・dc < 0)、'\' は右下がり (dr・dc > 0)。
(push)
(declare-const dr Int)
(declare-const dc Int)
(define-fun ab ((x Int)) Int (ite (< x 0) (- x) x))
(define-fun vx () Int (ab dc))
(define-fun vy () Int (* 2 (ab dr)))
; 0 = '-', 1 = '|', 2 = '/', 3 = '\'
(define-fun glyph () Int
  (ite (<= (* 5 vy) (* 2 vx)) 0
  (ite (<= (* 5 vx) (* 2 vy)) 1
  (ite (= (< dr 0) (> dc 0)) 2 3))))
(assert (not (and
  (=> (= glyph 0) (<= (* 5 vy) (* 2 vx)))
  (=> (= glyph 1) (and (<= (* 5 vx) (* 2 vy)) (not (= dr 0))))
  (=> (= glyph 2) (< (* dr dc) 0))
  (=> (= glyph 3) (> (* dr dc) 0)))))
(check-sat)
(pop)

; ---- 惑星と月 (internal/sky/moon.go・render.go)。実装を確かめるテスト:
;      TestSpec_MoonAge・TestSpec_BodyFits (internal/sky/planet_spec_test.go)。

; 7. 月齢と月の相: 太陽からの離角 e (0 以上 360 未満、norm360 の後) の月齢 age = e/360・S は 0 以上 S 未満。
;    相 p = floor(age/S・8 + 1/2) mod 8 は 0〜7 で、新月 (age = 0) は 0、満月 (age = S/2) は 4。
(push)
(declare-const e Real)
(declare-const k Int)
(define-fun S () Real 29.530588853)
(define-fun age () Real (* (/ e 360.0) S))
(assert (and (<= 0.0 e) (< e 360.0)))
; k = floor(age/S・8 + 1/2)
(assert (<= (to_real k) (+ (* (/ age S) 8.0) 0.5)))
(assert (< (+ (* (/ age S) 8.0) 0.5) (to_real (+ k 1))))
(assert (not (and (<= 0.0 age) (< age S)
                  (<= 0 (mod k 8)) (<= (mod k 8) 7)
                  (=> (= e 0.0) (= (mod k 8) 0))
                  (=> (= e 180.0) (= (mod k 8) 4)))))
(check-sat)
(pop)

; 8. bodyColumns: 天体の記号 (幅 n = 1 か 2) は、投影した列 col (0 <= col < w、w >= 2) か 1 つ左のどちらかに、
;    格子からはみ出さずに置ける (0 <= c・c + n <= w)。
(push)
(declare-const w Int)
(declare-const n Int)
(declare-const col Int)
(assert (and (>= w 2) (or (= n 1) (= n 2)) (<= 0 col) (< col w)))
(define-fun fits ((c Int)) Bool (and (<= 0 c) (<= (+ c n) w)))
(assert (not (or (fits col) (fits (- col 1)))))
(check-sat)
(pop)

; ---- 今夜の見どころ (internal/sky/tonight.go)。実装を確かめるテスト:
;      TestSpec_Compass16・TestSpec_HighlightLines (internal/sky/tonight_spec_test.go)。

; 9. compassIndex: 方位 a (0 以上 360 未満、norm360 の後) の番号 k = floor(a/22.5 + 1/2) は 0〜16 で、
;    方位は k・22.5 度から ±11.25 度以内。k = 16 (348.75 度以上) は 360 度 = 北で、mod 16 で 0 (北) になる。
;    だから 16 方位の名前は、いつも方位にいちばん近い 16 方位 (ずれは 11.25 度まで)。
(push)
(declare-const a Real)
(declare-const k Int)
(assert (and (<= 0.0 a) (< a 360.0)))
(assert (<= (to_real k) (+ (/ a 22.5) 0.5)))
(assert (< (+ (/ a 22.5) 0.5) (to_real (+ k 1))))
(assert (not (and (<= 0 k) (<= k 16)
                  (<= (- a (* 22.5 (to_real k))) 11.25)
                  (<= (- (* 22.5 (to_real k)) a) 11.25)
                  (<= 0 (mod k 16)) (<= (mod k 16) 15))))
(check-sat)
(pop)

; 10. highlights: 見どころは 2 行 (一番明るい惑星か「昇る惑星がない」・月齢と月の出入り) に、
;     接近があれば 1 行を足す。だから 1〜3 行に収まる。
(push)
(declare-const approach Bool)
(define-fun lines () Int (+ 2 (ite approach 1 0)))
(assert (not (and (<= 1 lines) (<= lines 3))))
(check-sat)
(pop)

; ---- 方角を向いた図 (internal/sky/facing.go)。実装を確かめるテスト:
;      TestSpec_FacingInside・TestSpec_RelAz (internal/sky/facing_test.go)。

; 11. facingView.pos: 図に入るもの (高度 alt が 0 より大きく 90 以下・向きからの角 rel が -90〜90) は、
;     行 row = round(t・bottom) (t = (90 - alt)/90、bottom = h - 2) が 0〜bottom、
;     列のずれ k = round(u・hx) (u = rel・cos(alt)/90、hx = div (w-1) 2) が -hx〜hx で、cx = hx なら列 cx + k は 0〜w-1。
;     11a: 0 <= t < 1、-1 <= u <= 1 (cos(alt) を 0〜1 の実数 c とおく)。
(push)
(declare-const alt Real)
(declare-const rel Real)
(declare-const c Real)
(assert (and (< 0.0 alt) (<= alt 90.0) (<= (- 90.0) rel) (<= rel 90.0) (<= 0.0 c) (<= c 1.0)))
(define-fun t () Real (/ (- 90.0 alt) 90.0))
(define-fun u () Real (/ (* rel c) 90.0))
(assert (not (and (<= 0.0 t) (< t 1.0) (<= (- 1.0) u) (<= u 1.0))))
(check-sat)
(pop)
;     11b: だから y = t・bottom は 0〜bottom、x = u・hx は -hx〜hx の実数 (3 と同じく範囲を前提にする)。
;     それを四捨五入した整数 (ずれ 1/2 以内) も同じ範囲に入り、0 <= cx - hx・cx + hx <= w - 1。
(push)
(declare-const w Int)
(declare-const h Int)
(declare-const y Real)
(declare-const x Real)
(declare-const row Int)
(declare-const k Int)
(assert (and (<= 20 w) (<= w 400) (<= 6 h) (<= h 199)))
(define-fun bottom () Int (- h 2))
(define-fun hx () Int (div (- w 1) 2))
(assert (and (<= 0.0 y) (<= y (to_real bottom)) (<= (- (to_real hx)) x) (<= x (to_real hx))))
(assert (and (<= (- (to_real row) y) 0.5) (<= (- y (to_real row)) 0.5)))
(assert (and (<= (- (to_real k) x) 0.5) (<= (- x (to_real k)) 0.5)))
(assert (not (and (<= 0 row) (<= row bottom) (<= (- hx) k) (<= k hx)
                  (>= bottom 4) (>= hx 9) (<= (+ hx hx) (- w 1)))))
(check-sat)
(pop)

; 12. relAz: 方位 az を向き center から見た角 r = norm360(az - center + 180) - 180 (norm360 は [0, 360)) は
;     -180 以上 180 未満で、az - center と 360 の倍数だけ違う。
;     2 点の r の差が 180 より大きいとき、そのあいだの近い方の弧は真後ろ (r = -180) を通る (線は描かない)。
(push)
(declare-const d Real) ; az - center + 180
(declare-const n Int)
(declare-const m Real) ; norm360(d) = d - 360・n
(assert (= m (- d (* 360.0 (to_real n)))))
(assert (and (<= 0.0 m) (< m 360.0)))
(define-fun r () Real (- m 180.0))
(assert (not (and (<= (- 180.0) r) (< r 180.0) (= (- (- d 180.0) r) (* 360.0 (to_real n))))))
(check-sat)
(pop)
(push)
(declare-const r1 Real)
(declare-const r2 Real)
(assert (and (<= (- 180.0) r1) (< r1 180.0) (<= (- 180.0) r2) (< r2 180.0) (<= r1 r2)))
; 差が 180 より大きければ、まっすぐ (r1 → r2) より、真後ろを通る方 (r2 → 180 = -180 → r1) が短い
(assert (> (- r2 r1) 180.0))
(assert (not (< (+ (- 180.0 r2) (+ r1 180.0)) (- r2 r1))))
(check-sat)
(pop)

; ---- 終了コード (internal/app/app.go)。実装を確かめるテスト: TestSpec_ExitCode (internal/app/exit_spec_test.go)。

; 13. ExitCode: 成功は 0、引数の誤り (打ち直せば通る) は 2、それ以外の失敗は 1。3 つは重ならず、どの結果にも 1 つ決まる。
(push)
(declare-const ok Bool)
(declare-const usage Bool)
(define-fun code () Int (ite ok 0 (ite usage 2 1)))
(assert (not (and (=> ok (= code 0))
                  (=> (and (not ok) usage) (= code 2))
                  (=> (and (not ok) (not usage)) (= code 1))
                  (or (= code 0) (= code 1) (= code 2)))))
(check-sat)
(pop)
