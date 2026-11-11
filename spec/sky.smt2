; terminal-sky の描画の仕様 (internal/sky/render.go)。
; 決まり: 証明したい性質の「否定」を assert し、(check-sat) が unsat なら証明できた。
; 実装を確かめるテスト: TestSpec_Layout・TestSpec_SymbolMonotonic・TestSpec_ProjectInside (internal/sky/spec_test.go)。

; 1. layout: 図の高さ h (7〜199、見出しの 1 行を除いた高さ) と幅 w (20〜400) のどれでも、
;    地平線の楕円 (中心 cx, cy・半径 rx, ry) が格子に収まり、半径は 1 以上。
;    Go の整数の割り算は、0 以上の数どうしでは div と同じ。
(push)
(declare-const w Int)
(declare-const h Int)
(assert (and (<= 20 w) (<= w 400) (<= 7 h) (<= h 199)))
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
