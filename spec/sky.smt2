; terminal-sky の描画の仕様 (internal/sky/render.go)。
; 決まり: 証明したい性質の「否定」を assert し、(check-sat) が unsat なら証明できた。
; 実装を確かめるテスト: TestSpec_Layout・TestSpec_SymbolMonotonic・TestSpec_ProjectInside・TestSpec_SegmentShown・
; TestSpec_LineInsideHorizon・TestSpec_LineGlyph (internal/sky/spec_test.go)、
; TestSpec_MoonAge・TestSpec_BodyFits (internal/sky/planet_spec_test.go)、
; TestSpec_Compass16・TestSpec_HighlightLines (internal/sky/tonight_spec_test.go)。

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
