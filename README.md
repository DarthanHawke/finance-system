# Transaction System
Микросервисная платежная система
> [!WARNING]
> Проект ещё находится в активной разработке и не все представленные функции реализованы( p.s. А ещё я все переделал, но не все доделал, хех. Пока сделал только: Account Service(содержит данные счета, управляет счетом), Transaction Service(проводит транакции по саге), External Payment Service(на коленке собранная эмуляция внешней платежки, которая взаиммодействует с Transaction), и собственно Api-gateway чтоб все это тыкать. С предыдущей итерации проекта еще не перенес авторизацию, аутентификацию и управление ролями)
Зато вот bpmn схемки саги транзакций: хореография всей саги и отдельно три схемки для каждого типа платежа

[![Схема транзакций](assets/Finance-service_choreography_diagram.bpmn.svg)](https://raw.githubusercontent.com/DarthanHawke/finance-system/5b0445f563ac2a49c54c8ec56c66bd31cee3aa68/assets/Finance-service_choreography_diagram.bpmn.svg)

[![C1](assets/Finance-service_C1_physical_diagram.png)](https://raw.githubusercontent.com/DarthanHawke/finance-system/refs/heads/develop/assets/Finance-service_C1_physical_diagram.png)

[![C2](assets/Finance-service_C2_physical_diagram.png)](https://raw.githubusercontent.com/DarthanHawke/finance-system/refs/heads/develop/assets/Finance-service_C1_physical_diagram.png)

[![C3](assets/Finance-service_C3_physical_diagram.png)](https://raw.githubusercontent.com/DarthanHawke/finance-system/refs/heads/develop/assets/Finance-service_C3_physical_diagram.png)