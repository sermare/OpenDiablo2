scenario_name="w44 vendors: permanent stock survives a purchase, poor hero gets the vendor's message, shift buy of potions, gamble restock via the regular window"
scenario_env() {
  echo 'export OD2_AUTOTRADE="Akara,Charsi" OD2_AUTOTRADE_SEED=1 OD2_AUTOTRADE_LEVEL=8'
}
scenario_check() {
  grep -E "AUTOTRADE (permanent|poor|multibuy)|vendor message" $log.txt | cut -c1-220
  # Akara always sells potions/scrolls: a purchase leaves the stock unchanged
  grep -qE "AUTOTRADE permanent vendor=Akara code=[a-z0-9]+ stock_before=([0-9]+) stock_after=\1 err=<nil>" $log.txt || { echo "FAIL: Akara's permanent item left the stock"; fail=1; }
  grep -qE "AUTOTRADE poor vendor=Akara err=not enough gold message=\"You do not have enough gold for that.\"" $log.txt || { echo "FAIL: no not-enough-gold message"; fail=1; }
  grep -qE "vendor message: code=0xc " $log.txt || { echo "FAIL: message not shown"; fail=1; }
  grep -qE "AUTOTRADE multibuy vendor=Akara code=[a-z0-9]+ unit=[0-9]+ paid=[0-9]+ copies=([3-9]|[1-9][0-9]) " $log.txt || { echo "FAIL: shift buy did not buy several potions"; fail=1; }
}
