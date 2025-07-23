# Proyecto del tercer sprint

Bases de datos, usar vitess con MySQL, enfocarnos en la performance y escalabilidad.

La idea es implementar un sistema que permita manejar grandes volúmenes de datos y consultas de forma descentralizada, optimizando el rendimiento y la escalabilidad del sistema.

posiblemente el proyecto sea un clon de Amazon o un e-commerce (nada de front ni cosas de las compras solamente el backend para hacer consultas).

la base de datos seguramente sea un modelo de tres o cuatro tablas.
productos, usuarios, pedidos, posiblemente una tabla de categorías y reviews.

vitess sirve para la gestión de bases de datos distribuidas, muy buena para escalar horizontalmente y manejar grandes volúmenes de datos de manera eficiente.
La idea es dividir las BDs por paises y usar vitess para gestionar las conexiones y consultas de manera eficiente.
vitess no es buena para la busqueda de texto completo asi que quiero integrar elasticsearch o opensearch.(ni idea como podria integrar esto)

igual estoy pensando como hacer consultas descentralizadas, o sea una sola consulta que se divide en distintas BDs


///////////////////////////////////////////////////////////////////////////////////
//podremos usar algo para analizar el rendimiento de la BD como openSearch o elasticsearch
V2

Centrarse en el escalado horizontal y ver que tanto se puede hacer con vitess y MySQL.
Podre hacer que vitess se conecte a otros vitess??? (cluster de vitess, parece que si)

///////////////////////////////////////////////////////////////////////////////////

prometheus para monitorear el rendimiento de las consultas y la base de datos. 

una red de servidores no centralizados que son gestionados por vitess, cada servidor tiene sus propias base de datos y vitess se encarga de gestionar las conexiones y consultas de manera eficiente.
Seguramente implemente prometheus para monitorear el rendimiento de las BD's y las consultas

Me estoy enfocando demasiado en la arquitectura de las bases de datos y no tanto en la eficiencia de las mismas