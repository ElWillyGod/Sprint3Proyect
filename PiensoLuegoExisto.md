# Proyecto del tercer sprint
usaremos go, postgres para la base de datos y uso de indexes complejos.
ademas de un cache en memoria con redis.

esto va a ser una aplicacion que controle productos de una tienda.
la idea es que se hagan consultas a una base de datos y cuando la cantidad de consultas superen un umbral, levantar otro contenedor y que se mantengan sincronizados entre ellos.
permitiendo asi escalar horizontalmente y balancear la carga entre los distintos contenedores.

cuando la cantidad de consultas disminuya por debajo de otro umbral, se puede reducir el número de contenedores.

y los cambios que se realicen en la base de datos se replican entre los distintos contenedores.
