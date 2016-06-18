package main

import (
	"bytes"
	"fmt"
	//"net/http"
	"crypto/md5"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	godotenv "github.com/joho/godotenv"
	"github.com/rockneurotiko/go-tgbot"

	"database/sql"
	"upper.io/db"
	"upper.io/db/mysql"
	"upper.io/db/util/sqlutil"
)

type Padron struct {
	Nombre       string `db:"nombre"`
	TelegramUID  string `db:"telegram_uid"`
	TelegramGID  string `db:"telegram_gid"`
	AliasSecreto string `db:"alias_secreto"`
}

type Votacion struct {
	Id      int      `db:"id"`
	Titulo  string   `db:"titulo"`
	Desc    string   `db:"descripcion"`
	Inicio  timeType `db:"inicio"`
	Fin     timeType `db:"fin"`
	Secreto string   `db:"secreto"`
	Estatus int      `db:"estatus"`
}

type Planilla struct {
	Id          int    `db:"id"`
	Id_votacion int    `db:"id_votacion"`
	Titular     string `db:"titular"`
	Desc        string `db:"descripcion"`
}

type Catalogo struct {
	Id     int    `db:"id"`
	Nombre string `db:"nombre"`
	Valor  string `db:"valor"`
	Tipo   string `db:"tipo"`
	Desc   string `db:"descripcion"`
}

type VotacionCat struct {
	Id_v        int    `db:"id_votacion"`
	Id_c        int    `db:"id_catalogo"`
	Significado string `db:"significado"`
}

type Voto struct {
	Id      int    `db:"id"`
	Id_v    int    `db:"id_votacion"`
	Id_p    int    `db:"id_planilla"`
	Valor   string `db:"valor"`
	Secreto string `db:"alias_secreto"`
}

type resultado struct {
	Valor    string `db:"valor"`
	Votos    int64  `db:"votos"`
	DeTantos int64  `db:"detantos"`
}

var commandArray []string
var availableCommands = map[string]string{
	"a hola":           "Bienvenida y avisos",
	"b ayuda":          "Ayuda de Comandos",
	"c ant":            "Votaciones anteriores",
	"d vig":            "Votaciones vigentes",
	"e exp <votacion>": "Explica la votación",
	"f bol <votacion>": "Ver boleta",
	"g vot <planilla>": "votar boleta",
	"h res <votacion>": "Resultados de votación",
}

// Struct for testing marshalling.
type timeType struct {
	// Time is handled internally as time.Time but saved
	// as an (integer) unix timestamp.
	value time.Time
}

// time.Time -> unix timestamp
func (u timeType) MarshalDB() (interface{}, error) {
	return u.value.Unix(), nil
}

// Note that we're using *timeType and no timeType.
// unix timestamp -> time.Time
func (u *timeType) UnmarshalDB(v interface{}) error {
	var i int

	switch t := v.(type) {
	case string:
		i, _ = strconv.Atoi(t)
	default:
		return db.ErrUnsupportedValue
	}

	t := time.Unix(int64(i), 0)
	*u = timeType{t}

	return nil

}

var sess db.Database
var drv *sql.DB

func main() {
	godotenv.Load("secrets.env")
	token := os.Getenv("TELEGRAM_KEY")

	var settings = mysql.ConnectionURL{
		Address:  db.Host("localhost"),
		Database: os.Getenv("DB"),
		User:     os.Getenv("DB_USR"),
		Password: os.Getenv("DB_PSW"),
	}

	s, err := db.Open(mysql.Adapter, settings)
	if err != nil {
		fmt.Printf("db.Open(): %q\n", err)
	}
	sess = s

	drv = sess.Driver().(*sql.DB)
	defer sess.Close()

	// arrange commands
	for k := range availableCommands {
		commandArray = append(commandArray, k)
	}
	sort.Strings(commandArray)

	bot := tgbot.NewTgBot(token).
		//AnyMsgFn(allMsgHand).
		SimpleRegexFn(`^/vig`, vigentesHandler).
		SimpleRegexFn(`^/ant`, noVigentesHandler).
		CommandFn(`^/exp (.+)`, explicaHandler).
		CommandFn(`^/bol (.+)`, boletaHandler).
		CommandFn(`^/vot (.+)`, votaHandler).
		CommandFn(`^/res (.+)`, resultadosHandler).
		SimpleRegexFn(`hola`, bienvenidaHandler).
		SimpleRegexFn(`start`, bienvenidaHandler).
		MultiCommandFn([]string{`ayuda (\w+)`, `ayuda`}, multiregexHelpHand).
		CustomFn(conditionFunc, conditionCallFunc)

	bot.StartChain().
		SimpleRegexFn(`^(cat|dog|nya|chick)$`, answer).
		CancelChainCommand(`cancel`, justtest).
		EndChain()

	//SimpleCommandFn(`keyboard`, cmdKeyboard).
	//SimpleCommandFn(`hidekeyboard`, hideKeyboard).
	//SimpleCommandFn(`forwardme`, forwardHand).
	//MultiCommandFn([]string{`help (\w+)`, `help`, `helpbotfather`}, multiregexHelpHand).
	//SimpleCommandFn(`senddocument`, sendDocument).
	//DocumentFn(returnDocument).
	//SimpleCommandFn(`sendsticker`, sendSticker).
	//StickerFn(returnSticker)

	bot.DefaultDisableWebpagePreview(true)      // Disable all link preview by default
	bot.DefaultOneTimeKeyboard(true)            // Enable one time keyboard by default
	bot.DefaultSelective(true)                  // Use Seletive by default
	bot.DefaultCleanInitialUsername(true)       // By default is true! (This removes initial @username from messages)
	bot.DefaultAllowWithoutSlashInMention(true) // By default is true! (This adds the / in the messages that have @username, this needs DefaultCleanInitialUsername true, for example: @username test becomes /test)

	bot.SimpleStart()

}

func vigentesHandler(bot tgbot.TgBot, msg tgbot.Message, text string) *string {
	collection, err := sess.Collection("votacion")
	if err != nil {
		fmt.Printf("sess.Collectoin(): %q\n", err)
	}

	var res db.Result
	res = collection.Find(db.And{
		db.Raw{"inicio <= now()"},
		db.Raw{"fin >= now()"},
	})
	var votacion []Votacion
	err = res.All(&votacion)
	if err != nil {
		fmt.Printf("res.All(): %q\n", err)
	}

	var buffer bytes.Buffer
	for _, v := range votacion {
		str := fmt.Sprintf("Votación: %d\n%s\n%s\n\n", v.Id, v.Titulo, v.Desc)
		buffer.WriteString(str)
	}

	//bot.Answer(msg).Text("Usa el telcado para comandos básicos").Keyboard(keyboard()).End()
	msgr := ""
	msgr = buffer.String()
	return &msgr
}

func noVigentesHandler(bot tgbot.TgBot, msg tgbot.Message, text string) *string {
	collection, err := sess.Collection("votacion")
	if err != nil {
		fmt.Printf("sess.Collectoin(): %q\n", err)
	}

	var res db.Result
	res = collection.Find(db.And{
		db.Raw{"inicio <= now()"},
		db.Raw{"fin <= now()"},
	})
	var votacion []Votacion
	err = res.All(&votacion)
	if err != nil {
		fmt.Printf("res.All(): %q\n", err)
	}

	var buffer bytes.Buffer
	for _, v := range votacion {
		str := fmt.Sprintf("Votación ya sufragada: %d\n%s\n\n", v.Id, v.Titulo)
		buffer.WriteString(str)
	}

	//bot.Answer(msg).Text("Usa el telcado para comandos básicos").Keyboard(keyboard()).End()
	msgr := ""
	msgr = buffer.String()
	return &msgr
}

func explicaHandler(bot tgbot.TgBot, msg tgbot.Message, vals []string, kvals map[string]string) *string {
	votacion_id := 0
	if len(vals) > 1 {
		votacion_id, _ = strconv.Atoi(vals[1])
	}

	collection, err := sess.Collection("votacion")
	if err != nil {
		fmt.Printf("sess.Collectoin(): %q\n", err)
	}

	var res db.Result
	res = collection.Find(db.Cond{"id": votacion_id})
	var v Votacion
	err = res.One(&v)
	if err != nil {
		fmt.Printf("res.One(): %q\n", err)
	}

	str := fmt.Sprintf("ID Votación: %d\n%s\n%s\n\n", v.Id, v.Titulo, v.Desc)
	return &str
}

func boletaHandler(bot tgbot.TgBot, msg tgbot.Message, vals []string, kvals map[string]string) *string {
	votacion_id := 0
	if len(vals) > 1 {
		votacion_id, _ = strconv.Atoi(vals[1])
	}

	collection, err := sess.Collection("planilla")
	if err != nil {
		fmt.Printf("sess.Collectoin(): %q\n", err)
	}

	var res db.Result
	res = collection.Find(db.Cond{"id_votacion": votacion_id})
	var boleta []Planilla
	err = res.All(&boleta)
	if err != nil {
		fmt.Printf("res.One(): %q\n", err)
	}

	var buffer bytes.Buffer
	for _, b := range boleta {
		str := fmt.Sprintf("ID:%d - %s\n%s\n\n", b.Id, b.Titular, b.Desc)
		buffer.WriteString(str)
	}
	msgr := ""
	msgr = buffer.String()
	return &msgr
}

func votaHandler(bot tgbot.TgBot, msg tgbot.Message, vals []string, kvals map[string]string) *string {
	var buffer bytes.Buffer
	var res db.Result
	var err error
	id_planilla := 0

	if len(vals) > 1 {
		id_planilla, _ = strconv.Atoi(vals[1])
	}

	coll_planilla, err := sess.Collection("planilla")
	if err != nil {
		fmt.Printf("sess.Collectoin(): %q\n", err)
	}
	coll_votacion, err := sess.Collection("votacion")
	if err != nil {
		fmt.Printf("sess.Collectoin(): %q\n", err)
	}

	//if padron(*msg.From.Username) {
	/**
	* Datos boleta
	 */
	res = coll_planilla.Find(db.Cond{"id": id_planilla})
	var boleta Planilla
	err = res.One(&boleta)
	if err != nil {
		//fmt.Printf("res.One(): %q\n", err)
		buffer.WriteString("ID de boleta no existe, intenta con uno existente")
	} else {

		/**
		* Datos votacion
		 */
		res = coll_votacion.Find(db.And{
			db.Cond{"id": boleta.Id_votacion},
			db.Raw{"inicio <= now()"},
			db.Raw{"fin >= now()"},
		})
		var votacion Votacion
		err = res.One(&votacion)
		if err != nil {
			//fmt.Printf("res.One(): %q\n", err)
			buffer.WriteString("Intentas votar una votación no vigente o inexistente")
		} else {
			/**
			* Se aplica el voto
			 */
			if col_voto, err := sess.Collection("voto"); err == nil {
				if msg.From.Username != nil {
					_, err = col_voto.Append(Voto{
						Id_p:    id_planilla,
						Id_v:    boleta.Id_votacion,
						Valor:   boleta.Titular,
						Secreto: oneWay(*msg.From.Username, votacion.Secreto),
					})
				} else {
					_, err = col_voto.Append(Voto{
						Id_p:    id_planilla,
						Id_v:    boleta.Id_votacion,
						Valor:   boleta.Titular,
						Secreto: oneWay(strconv.Itoa(msg.From.ID), votacion.Secreto),
					})
				}
				if err != nil {
					buffer.WriteString("Error, o ya aplicaste tu voto")
				} else {
					buffer.WriteString("Gracias por votar!")
				}
			} else {
				buffer.WriteString(fmt.Sprintf("res.One(): %q\n", err))
			}
		}
	}
	//} else {
	//		buffer.WriteString("Lo sentimos, tu usuario de telegram no está en el padrón. Si eres miembro de Wikipolítica CDMX solicita tu alta en el padrón de WikiVoto")
	//	}

	msgr := ""
	msgr = buffer.String()
	return &msgr
}

func resultadosHandler(bot tgbot.TgBot, msg tgbot.Message, vals []string, kvals map[string]string) *string {
	var buffer bytes.Buffer
	var rows *sql.Rows
	var resv db.Result

	var err error
	id_votacion := 0

	if len(vals) > 1 {
		id_votacion, _ = strconv.Atoi(vals[1])
	}

	//if padron(*msg.From.Username) {
	/**
	 * Calcula la votación por id votación
	 */
	qry := `
			SELECT valor, COUNT(valor) AS votos, (SELECT COUNT(*) FROM voto WHERE id_votacion = ?) AS detantos
			FROM voto va
			LEFT JOIN votacion AS v ON v.id = va.id_votacion
			WHERE va.id_votacion = ? AND
			v.fin <= now()
			GROUP BY valor
		`
	rows, err = drv.Query(qry, id_votacion, id_votacion)

	if err != nil {
		buffer.WriteString(fmt.Sprintf("Ooops! ocurrió un error. drv.Query(): %q\n", err))
	}
	var res []resultado

	// Mapping to an array.
	if err := sqlutil.FetchRows(rows, &res); err != nil {
		buffer.WriteString(fmt.Sprintf("Ooops! ocurrió un error. sqlutil.Fetchrows(): %q\n", err))
	}

	/**
	 * Datos votacion
	 */
	coll_votacion, err := sess.Collection("votacion")
	if err != nil {
		fmt.Printf("sess.Collectoin(): %q\n", err)
	}
	resv = coll_votacion.Find(db.And{
		db.Cond{"id": id_votacion},
		db.Raw{"fin <= now()"},
	})
	var votacion Votacion
	err = resv.One(&votacion)
	if err != nil {
		//fmt.Printf("res.One(): %q\n", err)
		buffer.WriteString("Intentas ver resultados de una votación *vigente* o inexistente")
	} else {
		// Imprime resultados
		buffer.WriteString(fmt.Sprintf("Resultados de la votacion %d\n%s\n%s\n", id_votacion, votacion.Titulo, votacion.Desc))
		if len(res) != 0 {
			for _, v := range res {
				str := fmt.Sprintf("%s: %d de un total de %d\n", v.Valor, v.Votos, v.DeTantos)
				buffer.WriteString(str)
			}
		} else {
			buffer.WriteString(fmt.Sprintf("No hay resultados para la votacion %d\n", id_votacion))
		}
	}
	//} else {
	//		buffer.WriteString("Lo sentimos, tu usuario de telegram no está en el padrón. Si eres miembro de Wikipolítica CDMX solicita tu alta en el padrón de WikiVoto")
	//}

	msgr := ""
	msgr = buffer.String()
	return &msgr
}

func bienvenidaHandler(bot tgbot.TgBot, msg tgbot.Message, text string) *string {
	var str string
	//if padron(*msg.From.Username) {
	str = "Bienvenido al Bot de votaciones de WikiCDMX\nFeliz votación :)\nEscribe /ayuda para iniciar"
	//} else {
	//str = "Lo sentimos, tu usuario de telegram no está en el padrón. Si eres miembro de Wikipolítica CDMX solicita tu alta en el padrón de WikiVoto"
	//}
	msgr := fmt.Sprintf("Hola %s! <3\n%s", msg.From.FirstName, str)
	return &msgr
}

func multiregexHelpHand(bot tgbot.TgBot, msg tgbot.Message, vals []string, kvals map[string]string) *string {
	if len(vals) > 1 {
		for k, v := range availableCommands {
			if k[1:] == vals[1] {
				res := v
				return &res
			}
		}
	}
	res := ""
	if vals[0] == "/ayuda" {
		res = buildHelpMessage(true)
	}
	//bot.Answer(msg).Text("Comandos básicos al teclado").Keyboard(keyboard()).End()
	return &res
}

func allMsgHand(bot tgbot.TgBot, msg tgbot.Message) {
	// uncomment this to see it :)
	fmt.Printf("Received message: %+v\n", msg)
	// bot.SimpleSendMessage(msg, "Received message!")
}

func conditionFunc(bot tgbot.TgBot, msg tgbot.Message) bool {
	return msg.Photo != nil
}

func conditionCallFunc(bot tgbot.TgBot, msg tgbot.Message) {
	fmt.Printf("Text: %+v\n", msg.Text)
	// bot.SimpleSendMessage(msg, "Nice image :)")
}

func sendDocument(bot tgbot.TgBot, msg tgbot.Message, text string) *string {
	mid := msg.ID
	bot.Answer(msg).
		Document("example/simpleexample/files/PracticalPrincipledFRP.pdf").
		ReplyToMessage(mid).
		End()
	// bot.SendDocument(msg.Chat.ID, "example/simpleexample/files/PracticalPrincipledFRP.pdf", &mid, nil)
	return nil
}

func returnDocument(bot tgbot.TgBot, msg tgbot.Message, document tgbot.Document, fid string) {
	bot.Answer(msg).
		Document(fid).
		End()
	// bot.SimpleSendDocument(msg, fid)
}

func sendSticker(bot tgbot.TgBot, msg tgbot.Message, text string) *string {
	bot.Answer(msg).Sticker("example/simpleexample/files/sticker.webp").End()
	// bot.SimpleSendSticker(msg, "example/simpleexample/files/sticker.webp")
	return nil
}

func returnSticker(bot tgbot.TgBot, msg tgbot.Message, sticker tgbot.Sticker, fid string) {
	mid := msg.ID
	bot.Answer(msg).
		Sticker(fid).
		ReplyToMessage(mid).
		End()
	// bot.SendSticker(msg.Chat.ID, fid, &mid, nil)
}

func answer(bot tgbot.TgBot, msg tgbot.Message, text string) *string {
	mytext := "Comando no reconocido"
	return &mytext
}

func justtest(bot tgbot.TgBot, msg tgbot.Message, text string) *string {
	return &text
}

func hideKeyboard(bot tgbot.TgBot, msg tgbot.Message, text string) *string {
	rkm := tgbot.ReplyKeyboardHide{HideKeyboard: true, Selective: false}
	bot.Answer(msg).Text("Hidden it!").KeyboardHide(rkm).End()
	// bot.SendMessageWithKeyboardHide(msg.Chat.ID, "Hiden it!", nil, nil, rkm)
	return nil
}

/**
 * Helpers
 */
func buildHelpMessage(complete bool) string {
	var buffer bytes.Buffer
	buffer.WriteString("Para ejecutar un comando usa diagonal, p. ej. /ayuda.\n")
	//buffer.WriteString("Cuando sea posible te aparecerá un teclado de opciones o comandos directos.\n")
	i := 1
	for _, cmd := range commandArray {
		str := ""
		if complete {
			str = fmt.Sprintf("%d) %s: %s\n", i, cmd[2:], availableCommands[cmd])
		} else if len(strings.Split(cmd, "<")) > 1 {
			str = fmt.Sprintf("%d) %s: %s\n", i, cmd[1:], availableCommands[cmd])
		}
		buffer.WriteString(str)
		i++
	}
	return buffer.String()
}

func boletaKeyboard(boleta []Planilla) tgbot.ReplyKeyboardMarkup {
	keylayout := [][]string{{}}
	for _, b := range boleta {
		if len(keylayout[len(keylayout)-1]) == 2 {
			keylayout = append(keylayout, []string{b.Titular})
		} else {
			keylayout[len(keylayout)-1] = append(keylayout[len(keylayout)-1], b.Titular)
		}
	}

	rkm := tgbot.ReplyKeyboardMarkup{
		Keyboard:        keylayout,
		ResizeKeyboard:  false,
		OneTimeKeyboard: false,
		Selective:       false}
	return rkm

}

func keyboard() tgbot.ReplyKeyboardMarkup {
	keylayout := [][]string{{}}
	for _, k := range commandArray {
		if len(strings.Split(k[2:], " ")) == 1 {
			if len(keylayout[len(keylayout)-1]) == 2 {
				keylayout = append(keylayout, []string{k[2:]})
			} else {
				keylayout[len(keylayout)-1] = append(keylayout[len(keylayout)-1], fmt.Sprintf("/%s", k[2:]))
			}
		}
	}

	rkm := tgbot.ReplyKeyboardMarkup{
		Keyboard:        keylayout,
		ResizeKeyboard:  false,
		OneTimeKeyboard: false,
		Selective:       false}
	return rkm

}

func padron(uid string) bool {
	/*
		collection, err := sess.Collection("padron")
		if err != nil {
			fmt.Printf("sess.Collectoin(): %q\n", err)
		}
		var res db.Result
		res = collection.Find(db.Cond{"telegram_uid": uid})
		var p Padron
		err = res.One(&p)
		if err != nil {
			fmt.Printf("res.One(): %q\n", err)
			return false
		}
	*/
	return true
}

func oneWay(id string, secret string) string {
	h := md5.New()
	io.WriteString(h, secret)

	pwmd5 := fmt.Sprintf("%x", h.Sum(nil))

	// Specify two salt: salt1 = @#$% salt2 = ^&*()
	salt1 := "@#$%lkjhasdfuiwer237u/&h"
	salt2 := "^(/&$%LKJHoiTRY=)54()&*()"

	// salt1 + username + salt2 + MD5 splicing
	io.WriteString(h, salt1)
	io.WriteString(h, id)
	io.WriteString(h, salt2)
	io.WriteString(h, pwmd5)

	return fmt.Sprintf("%x", h.Sum(nil))
}
