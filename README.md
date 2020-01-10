# Server Info Query
Prueba presentada a Truora

## Contenido:
* *main.go*: Archivo principal para correr la api.
* *helper.go*: Contiene métodos útiles para imprimir en la consola de la aplicación los llamados a la api. 
* *server-info-api*: Archivo ejecutable de servidor web de la api.
* *serverinfo.apk*: Archivo de aplicación para Android.
* */images*: Carpeta que contiene  recortes del diseño de la aplicación.

## Pre Condiciones
Antes de realizar la consulta del dominio, se revisa que éste sea válido mediante el uso de una expresión regular. Si el usuario llama a la API con un dominio inválido, ésta devuelve un http status `400 Bad Request`. Se muestran ejemplos de dominios válidos e inválidos a continuación.

#### Válidos
* domain.com
* example.domain.com
* example.domain-hyphen.com
* www.domain.com
* example.museum

#### Inválidos
* http://example.com
* subdomain.-example.com
* example.com/parameter
* example.com?anything

## Documentación de la API
1. http://192.34.63.138:8005/serverinfo/${domain}

## 
	{
        “servers”:[
            {
                “address”: “ipAddress”,
                “ssl_grade”: “A+”,
                “country”: “US”,
                “owner”:  “Amazon, Inc.”
            }
        ],
        “servers_changed”: false,
        “ssl_grade”: “A+”,
        “previous_ssl_grade”: “”,
        “logo”: ”https://example.com/logo”,
        “title”: “Title”,
        “is_down”: false
    }

* *Servers*:
	* *address*: dirección IP obtenida del llamado a la api de SSL Labs.
	* *ssl_grade*: grado del certificado SSL del servidor calificado por SSL Labs, obtenido del llamado a su API.
	* *country*: país en el que se localiza el servidor. Dato obtenido de correr el comando _whois_ con la dirección IP del servidor obtenida de la api de SSL Labs. 
	* *country*: compañía dueña  del servidor. Dato obtenido de correr el comando _whois_ con la dirección IP del servidor obtenida de la api de SSL Labs. 
* *servers_changed*: Para obtener este campo, se guarda el registro de la consulta en la base de datos junto con la fecha y hora de creación. Cada vez que un usuario busca un dominio, primero se consulta en la base de datos para ver si ya se ha consultado antes, y de ser así, se revisa si la fecha de creación o modificación fue hace 1 hora o más.  Si se cumple esta condición, se extrae el registro de la base de datos y se vuelva a hacer un llamado a la API para comparar la información de los servidores. 
* *ssl_grade*”: El menor grado de SSL de todos los servidores encontrados. 
* *previous_ ssl_grade*: Se obtiene del registro de la base de datos como el actual *ssl_grade*, y si éste cambió con respecto a la nueva consulta, se convierte en este campo.
* *logo*: Se realiza un llamado a la página web del dominio ingresado y se obtiene  el elemento html `<Link>` que contenga un atributo que tenga un valor que contenga la palabra `icon`.  Ya que no todas las páginas web definen su logo de la misma manera, se revisa dos tipos de atributos que puede contener este valor.
* *title*: Se realiza un llamado a la página web del dominio ingresado y se obtiene el elemento html `<Title>` . 
* *is_down*: Se realiza un llamado a la página web del dominio ingresado y se revisa el http status de la respuesta, si este es ` 503 Service Unavailable` este campo es true.

2. http://192.34.63.138:8005/servers
## 
    { “items”: [
        { “domain”: “example.com”, 
        “info”: 
            {
            “servers”:[
                {
                    “address”: “ipAddress”,
                    “ssl_grade”: “A+”,
                    “country”: “US”,
                    “owner”:  “Amazon, Inc.”
                }
            ],
                “servers_changed”: false,
                “ssl_grade”: “A+”,
                “previous_ssl_grade”: “”,
                “logo”: ”https://example.com/logo”,
                “title”: “Title”,
                “is_down”: false
            }
        ]
    }

## Diagrama de Despliegue

![](https://raw.githubusercontent.com/PeppermintNDaisies/serverInfo/master/images/UMLDeploymentDiagram.png?token=AGTNIHHINUM4FL2JSI4KSTK6EEHKO)

## Diagrama de Modelo de Datos
![](https://raw.githubusercontent.com/PeppermintNDaisies/serverInfo/master/images/DataModel.png?token=AGTNIHH7LXBBK3UPQQG76TS6EEHHG)


## Diseño de Prototipo Digital
Pantalla principal
![](https://raw.githubusercontent.com/PeppermintNDaisies/serverInfo/master/images/app1.png?token=AGTNIHFICSLFUUNBFSQZYDC6EEHMG)

Revisión del dominio ingresado por el usuario
![](https://raw.githubusercontent.com/PeppermintNDaisies/serverInfo/master/images/app2.png?token=AGTNIHCS2POUGAV2EMAJEV26EEHMO)




