package main

import (
	"flag"
	"fmt"
	"github.com/gocolly/colly"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Date struct {
	Day string
	Month string
	Year string
}

type DailyMenu struct {
	RestaurantName string
	MenuDish []MenuDish
}

type MenuDish struct {
	Type string
	Name string
	Price string
}

type MyResultMenu struct {
	PivniceUCapa DailyMenu
	SuziesSteakPub DailyMenu
	VeroniCafe DailyMenu
}

func main() {
	dateArg := flag.String("date", "", "Target date in DD.MM.YYYY format")
	flag.Parse()

	matched, err := regexp.MatchString(`[1-9]{1,2}.[1-9]{1,2}.\d{2}(?:\d{2})?`, *dateArg)
	if err != nil || !matched {
		log.Fatal("Bad argument given for target date. Correct format is DD.MM.YYYY")
	}

	splittedDate := strings.Split(*dateArg, ".")
	formattedDate := splittedDate[2] + "-" + splittedDate[1] + "-" + splittedDate[0]
	targetDate, err := time.Parse("2006-1-2", formattedDate)
	if err != nil {
		log.Fatal("Error parsing target date format")
	}

	targetRestaurants := map[string]string{
		"PivniceUCapa": "https://www.pivnice-ucapa.cz/denni-menu.php",
		"SuziesSteakPub": "http://www.suzies.cz/poledni-menu.html",
		"VeroniCafe": "https://www.menicka.cz/4921-veroni-coffee--chocolate.html",
	}

	myResultMenu := MyResultMenu{}

	collector := GetCollectorInstanceForUrl()

	collector.OnHTML("div.listek", func(element *colly.HTMLElement) {
		myResultMenu.PivniceUCapa = ParsePivniceUCapaMenu(element, targetDate)
	})
	collector.OnHTML("div#weekly-menu", func(element *colly.HTMLElement) {
		myResultMenu.SuziesSteakPub = ParseSuziesSteakPubMenu(element, targetDate)
	})
	collector.OnHTML(".obsah", func(element *colly.HTMLElement) {
		myResultMenu.VeroniCafe = ParseVeroniCafeMenu(element, targetDate)
	})

	for _, restaurantUrl := range targetRestaurants{
		err := collector.Visit(restaurantUrl)
		if err != nil {
			log.Fatal("Could not visit we page: " + restaurantUrl)
		}
	}

	collector.Wait()

	RenderMenu(myResultMenu, targetDate)
}

func ParsePivniceUCapaMenu(element *colly.HTMLElement, targetDate time.Time) DailyMenu {
	dailyMenu := DailyMenu{RestaurantName: "Pivnice u Capa"}
	dishNameRegex := regexp.MustCompile(`[1-9]*\.(.*)`)

	element.ForEach("div.listek > div", func(i int, element *colly.HTMLElement) {
		splittedDateFromMenu := strings.Split(strings.ReplaceAll(element.ChildText(".date"), " ", ""), ".")

		parsedDate := Date{
			splittedDateFromMenu[0],
			splittedDateFromMenu[1],
			splittedDateFromMenu[2],
		}

		currentMenuDate, err := time.Parse("2006-1-2", parsedDate.Year + "-" + parsedDate.Month + "-" + parsedDate.Day)
		if err != nil {
			log.Fatal("Unable to parse date.")
		}

		if DateEqual(targetDate, currentMenuDate) {
			var dailyMenuDishes []MenuDish

			//add found soup
			dailyMenuDishes = append(dailyMenuDishes, MenuDish{
				Type: "Soup",
				Name: strings.TrimSpace(element.ChildText(".row-polevka > .polevka")),
			})

			//iterate through other dishes
			element.ForEach(".row-food", func(i int, element *colly.HTMLElement) {
				dailyMenuDishes = append(dailyMenuDishes, MenuDish{
					Type: "Main food",
					Name: strings.TrimSpace(dishNameRegex.FindAllStringSubmatch(element.ChildText(".food"), 1)[0][1]),
					Price: element.ChildText(".price"),
				})
			})
			dailyMenu.MenuDish = dailyMenuDishes
		}
	})

	return dailyMenu
}

func ParseSuziesSteakPubMenu(element *colly.HTMLElement, targetDate time.Time) DailyMenu {
	dailyMenu := DailyMenu{RestaurantName: "Suzies Steak Pub"}
	dateRegex := regexp.MustCompile(`.*\s([1-9]{1,2}\.[1-9]{1,2}\.)`)

	element.ForEach(".day", func(i int, element *colly.HTMLElement) {
		dateFromMenu := dateRegex.FindAllStringSubmatch(element.ChildText("h4"), 1)[0][1] + strconv.Itoa(time.Now().Year())
		splittedDateFromMenu := strings.Split(dateFromMenu, ".")

		parsedDate := Date{
			splittedDateFromMenu[0],
			splittedDateFromMenu[1],
			splittedDateFromMenu[2],
		}

		currentMenuDate, err := time.Parse("2006-1-2", parsedDate.Year + "-" + parsedDate.Month + "-" + parsedDate.Day)
		if err != nil {
			log.Fatal("Unable to parse date.")
		}

		if DateEqual(targetDate, currentMenuDate) {
			var dailyMainDishes []MenuDish
			element.ForEach(".item", func(i int, element *colly.HTMLElement) {
				category := strings.TrimSpace(element.DOM.Find(".category").Text())
				if category == "Polévka" {
					dailyMainDishes = append(dailyMainDishes, MenuDish{
						Type: "Soup",
						Name: strings.TrimSpace(element.ChildText(".title")),
					})
					return
				}

				dailyMainDishes = append(dailyMainDishes, MenuDish{
					Type: "Main food",
					Name: category + " " + strings.TrimSpace(element.ChildText(".title")) + ": " + strings.TrimSpace(element.ChildText(".text")),
					Price: strings.TrimSpace(element.ChildText(".price") + " Kč"),
				})
			})
			dailyMenu.MenuDish = dailyMainDishes
		}
	})

	return dailyMenu
}

func ParseVeroniCafeMenu(element *colly.HTMLElement, targetDate time.Time) DailyMenu {
	dailyMenu := DailyMenu{RestaurantName: "Veroni Cafe"}
	dateRegex := regexp.MustCompile(`.*\s([1-9]{1,2}\.[1-9]{1,2}\.\d{2}(?:\d{2})?)`)
	dishNameRegex := regexp.MustCompile(`[1-9]*\.(.*)`)

	element.ForEach(".menicka", func(i int, element *colly.HTMLElement) {
		dateFromMenu := dateRegex.FindAllStringSubmatch(element.ChildText("div.nadpis"), 1)
		splittedDateFromMenu := strings.Split(dateFromMenu[0][1], ".")

		parsedDate := Date{
			splittedDateFromMenu[0],
			splittedDateFromMenu[1],
			splittedDateFromMenu[2],
		}

		currentMenuDate, err := time.Parse("2006-1-2", parsedDate.Year + "-" + parsedDate.Month + "-" + parsedDate.Day)
		if err != nil {
			log.Fatal("Unable to parse date.")
		}

		if DateEqual(targetDate, currentMenuDate) {
			var dailyMainDishes []MenuDish
			element.ForEach("ul li.polevka", func(i int, element *colly.HTMLElement) {
				dailyMainDishes = append(dailyMainDishes, MenuDish{
					Type: "Soup",
					Name: strings.TrimSpace(element.ChildText(".polozka")),
					Price: strings.TrimSpace(element.ChildText(".cena")),
				})
			})

			element.ForEach("ul li.jidlo", func(i int, element *colly.HTMLElement) {
				dailyMainDishes = append(dailyMainDishes, MenuDish{
					Type: "Main food",
					Name: strings.TrimSpace(dishNameRegex.FindAllStringSubmatch(element.ChildText(".polozka"), 1)[0][1]),
					Price: strings.TrimSpace(	element.ChildText(".cena")),
				})
			})
			dailyMenu.MenuDish = dailyMainDishes
		}
	})

	return dailyMenu
}

//Checks if two dates are equal
func DateEqual(date1, date2 time.Time) bool {
	y1, m1, d1 := date1.Date()
	y2, m2, d2 := date2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

func GetCollectorInstanceForUrl() *colly.Collector  {
	collector := colly.NewCollector(
		//colly.Async(true),
		colly.MaxDepth(1),
	)

	collector.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: 3})

	collector.OnRequest(func(request *colly.Request) {
		fmt.Println("Visiting", request.URL.String())
	})

	collector.OnError(func(response *colly.Response, err error) {
		log.Println("Something went wrong:", err)
	})

	return collector
}

//Renders all structures and menus to CLI console as text
func RenderMenu(menu MyResultMenu, targetDate time.Time)  {
	fmt.Println("\nYour menu for you favorite restaurants for " + targetDate.Month().String() + " " + strconv.Itoa(targetDate.Day()) + "." + strconv.Itoa(targetDate.Year()))

	RenderRestaurantMenu(menu.PivniceUCapa)
	RenderRestaurantMenu(menu.SuziesSteakPub)
	RenderRestaurantMenu(menu.VeroniCafe)
}

func RenderRestaurantMenu(menu DailyMenu) {
	fmt.Println("")
	fmt.Println("")

	fmt.Println(menu.RestaurantName)
	for _, menuDish := range menu.MenuDish {
		if menuDish.Type == "Soup" {
			fmt.Println("Soup: " + menuDish.Name + " - " + menuDish.Price)
			continue
		}

		fmt.Println("Main food: " + menuDish.Name + " - " + menuDish.Price)
	}
}
