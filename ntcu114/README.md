# NTCU 114級 畢業專題

## release 與 allocationAsk 對不上
因為 pod 要做完才會 release ，但假設 pod 是 service 的話，就不會 release ，導致須排程的數目無法下降。
改成取出 pending 後成功解決這個問題。