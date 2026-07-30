package main

import (
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/starfederation/datastar-go/datastar"
	"golang.org/x/net/html"
)

func feedsEnrichHTMLOutput(htmlStr string) (string, error) {

	addDataAttributes := func(doc *html.Node) *html.Node {

		var walk func(*html.Node)

		count := 0

		walk = func(n *html.Node) {

			count++

			fmt.Println("walk(n)")

			if n.Type == html.ElementNode {

				var blockID string

				for _, attr := range n.Attr {
					fmt.Println(attr)
					if attr.Key == "data-block-id" {
						blockID = attr.Val
						break
					}
				}

				if blockID != "" {
					fmt.Println(blockID)
					n.Attr = append(n.Attr, html.Attribute{
						Key: "data-on:click",
						Val: datastar.GetSSE("/url/%s", blockID),
					})
				}

			}

			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walk(c)
			}

		}

		walk(doc)

		fmt.Println(count)

		return doc

	}

	removeOuterHTMLShell := func(doc *html.Node) (*html.Node, error) {

		var walk func(*html.Node)

		walk = func(n *html.Node) {
			// if doc != nil {
			// 	return
			// }

			if n.Type == html.ElementNode && n.Data == "body" {
				doc = n
				return
			}

			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walk(c)
			}
		}

		walk(doc)

		if doc == nil {
			return nil, fmt.Errorf("body element not found")
		}

		article := &html.Node{
			Type: html.ElementNode,
			Data: "article",
		}

		// Move every child from <body> into <article>.
		for doc.FirstChild != nil {
			child := doc.FirstChild
			doc.RemoveChild(child)
			article.AppendChild(child)
		}

		return article, nil
	}

	htmlNodes, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return "", err
	}

	htmlNodes = addDataAttributes(htmlNodes)

	// transformed, err := feedsStringifyHTML(htmlNodes)
	// if err != nil {
	// 	return "", err
	// }

	// fmt.Println(transformed)

	htmlNodes, err = removeOuterHTMLShell(htmlNodes)
	if err != nil {
		return "", err
	}

	transformed, err := feedsStringifyHTML(htmlNodes)
	if err != nil {
		return "", err
	}

	return transformed, nil

}

func feedsStringifyHTML(doc *html.Node) (string, error) {

	var b strings.Builder

	err := html.Render(&b, doc)

	if err != nil {
		return "", err
	}

	return b.String(), nil
}

func main() {

	input := `<article><p data-block-id="0">Imagine you are world hegemon and some shitty little country sidles up to you with an idea of making war on some other country. How can you tell if this is a bad idea? I mean, it’s already a bad idea, how can you assess <em><strong>how</strong></em> bad an idea it might be? Despite all the nonsense sprouted in thinktanks there are very old fashioned measures of warfighting ability which have significant predictive power. Consider a very simple one, which was standard for military planners back when it was a meaningful role: <a href="https://en.wikipedia.org/wiki/List_of_countries_by_steel_production">tons of steel</a> per year:<br>
(Millions of tons per year by year):</p><table data-block-id="1"><tbody><tr><td><span><span><a href="https://en.wikipedia.org/wiki/China"><img src="https://upload.wikimedia.org/wikipedia/commons/thumb/f/fa/Flag_of_the_People%27s_Republic_of_China.svg/40px-Flag_of_the_People%27s_Republic_of_China.svg.png" alt="China"></a></span></span><a href="https://en.wikipedia.org/wiki/Steel_industry_in_China">China</a></td><td>960.8</td><td>1,005.1</td><td>1,028.9</td><td>1,018.0</td><td>1,035.2</td><td>1,064.8</td><td>995.4</td><td>920.0</td><td>831.7</td><td>786.9</td><td>803.8</td></tr><tr><td><span><span><a href="https://en.wikipedia.org/wiki/India"><img src="https://upload.wikimedia.org/wikipedia/en/thumb/4/41/Flag_of_India.svg/40px-Flag_of_India.svg.png" alt="India"></a></span></span><a href="https://en.wikipedia.org/wiki/Iron_and_steel_industry_in_India">India</a></td><td>164.9</td><td>149.4</td><td>140.8</td><td>125.3</td><td>118.2</td><td>100.3</td><td>111.4</td><td>109.3</td><td>101.5</td><td>95.5</td><td>89.6</td></tr><tr><td><span><span><a href="https://en.wikipedia.org/wiki/United_States"><img src="https://upload.wikimedia.org/wikipedia/en/thumb/a/a4/Flag_of_the_United_States.svg/40px-Flag_of_the_United_States.svg.png" alt="United States"></a></span></span><a href="https://en.wikipedia.org/wiki/Iron_and_steel_industry_in_the_United_States">United States</a></td><td>82.0</td><td>79.5</td><td>81.4</td><td>80.5</td><td>85.8</td><td>72.7</td><td>87.8</td><td>86.6</td><td>81.6</td><td>78.5</td><td>78.9</td></tr><tr><td><span><span><a href="https://en.wikipedia.org/wiki/Japan"><img src="https://upload.wikimedia.org/wikipedia/en/thumb/9/9e/Flag_of_Japan.svg/40px-Flag_of_Japan.svg.png" alt="Japan"></a></span></span><a href="https://en.wikipedia.org/wiki/Japan">Japan</a></td><td>80.7</td><td>84.0</td><td>87.0</td><td>89.2</td><td>96.3</td><td>83.2</td><td>99.3</td><td>104.3</td><td>104.7</td><td>104.8</td><td>105.2</td></tr><tr><td><span><span><a href="https://en.wikipedia.org/wiki/Russia"><img src="https://upload.wikimedia.org/wikipedia/en/thumb/f/f3/Flag_of_Russia.svg/40px-Flag_of_Russia.svg.png" alt="Russia"></a></span></span><a href="https://en.wikipedia.org/wiki/Metallurgy_of_Russia">Russia</a></td><td>67.8</td><td>71.0</td><td>76.0</td><td>71.5</td><td>77.0</td><td>71.6</td><td>71.7</td><td>72.0</td><td>71.3</td><td>70.5</td><td>71.1</td></tr><tr><td><span><span><a href="https://en.wikipedia.org/wiki/South_Korea"><img src="https://upload.wikimedia.org/wikipedia/commons/thumb/0/09/Flag_of_South_Korea.svg/40px-Flag_of_South_Korea.svg.png" alt="South Korea"></a></span></span><a href="https://en.wikipedia.org/wiki/South_Korea">South Korea</a></td><td>61.9</td><td>63.6</td><td>66.7</td><td>65.8</td><td>70.4</td><td>67.1</td><td>71.4</td><td>72.5</td><td>71.1</td><td>68.6</td><td>69.7</td></tr><tr><td><span><span><a href="https://en.wikipedia.org/wiki/Turkey"><img src="https://upload.wikimedia.org/wikipedia/commons/thumb/b/b4/Flag_of_Turkey.svg/40px-Flag_of_Turkey.svg.png" alt="Turkey"></a></span></span><a href="https://en.wikipedia.org/wiki/Turkey">Turkey</a></td><td>38.1</td><td>36.9</td><td>33.7</td><td>35.1</td><td>40.4</td><td>35.8</td><td>33.7</td><td>37.3</td><td>37.5</td><td>33.2</td><td>31.5</td></tr><tr><td><span><span><a href="https://en.wikipedia.org/wiki/Germany"><img src="https://upload.wikimedia.org/wikipedia/en/thumb/b/ba/Flag_of_Germany.svg/40px-Flag_of_Germany.svg.png" alt="Germany"></a></span></span><a href="https://en.wikipedia.org/wiki/Germany">Germany</a></td><td>34.1</td><td>37.3</td><td>35.4</td><td>36.9</td><td>40.2</td><td>35.7</td><td>39.6</td><td>42.4</td><td>43.6</td><td>42.1</td><td>42.7</td></tr><tr><td><span><span><a href="https://en.wikipedia.org/wiki/Brazil"><img src="https://upload.wikimedia.org/wikipedia/en/thumb/0/05/Flag_of_Brazil.svg/40px-Flag_of_Brazil.svg.png" alt="Brazil"></a></span></span><a href="https://en.wikipedia.org/wiki/Brazil">Brazil</a></td><td>33.3</td><td>33.9</td><td>32.0</td><td>34.1</td><td>36.1</td><td>31.4</td><td>32.6</td><td>35.4</td><td>34.4</td><td>30.2</td><td>33.3</td></tr><tr><td><span><span><a href="https://en.wikipedia.org/wiki/Iran"><img src="https://upload.wikimedia.org/wikipedia/commons/thumb/c/ca/Flag_of_Iran.svg/40px-Flag_of_Iran.svg.png" alt="Iran"></a></span></span><a href="https://en.wikipedia.org/wiki/Iran">Iran</a></td><td>31.8</td><td>31.4</td><td>30.7</td><td>30.6</td><td>28.3</td><td>29.0</td><td>25.6</td><td>24.5</td><td>21.8</td><td>17.9</td><td>16.1</td></tr><tr><td><span><span><a href="https://en.wikipedia.org/wiki/Vietnam"><img src="https://upload.wikimedia.org/wikipedia/commons/thumb/2/21/Flag_of_Vietnam.svg/40px-Flag_of_Vietnam.svg.png" alt="Vietnam"></a></span></span><a href="https://en.wikipedia.org/wiki/Vietnam">Vietnam</a></td><td>24.7</td><td>22.0</td><td>19.2</td><td>20.0</td><td>23.0</td><td>19.9</td><td>17.5</td><td>15.5</td><td>10.3</td><td>7.8</td><td>5.7</td></tr><tr><td><span><span><a href="https://en.wikipedia.org/wiki/Italy"><img src="https://upload.wikimedia.org/wikipedia/en/thumb/0/03/Flag_of_Italy.svg/40px-Flag_of_Italy.svg.png" alt="Italy"></a></span></span><a href="https://en.wikipedia.org/wiki/Steel_industry_in_Italy">Italy</a></td><td>20.7</td><td>20.0</td><td>21.1</td><td>21.6</td><td>24.4</td><td>20.4</td><td>23.2</td><td>24.5</td><td>24.0</td><td>23.3</td><td>22.0</td></tr></tbody></table><p data-block-id="2">Steel is a good proxy more or less because you need to make things out of steel to blow up your enemies. It’s not perfect and has externalities. South Korea is going to have a hard time making as much steel without cheap oil from the middle east, coal and iron ore from Australia. Same with Japan. The US, Russia, Iran and Turkey don’t have this problem, they get all their raw materials locally, or can anyway. None the less, Japan gave the US a pretty good fight back in the day: if you make a lot of steel, even from imports that can get shut down, you have a lot of steel to deliver to your enemies in the form of bombs. Arguably same story with Germany, though they had historically decent local-ish supplies of coal and iron ore.</p><p data-block-id="3">You could make this per-capita and it probably wouldn’t change much. Steel represents more than the ultimate tonnage of shit you can drop on the enemy’s head: it also represents a form of social/industrial organization which doesn’t exist in places that don’t make so much steel. The level of industrial organization required to make 30+ million tons of steel a year is probably indicative of the ability to produce a shitload of drones or rockets or whatever (fighter planes are cool, but ridiculously expensive and of only moderate utility in current conditions).</p><p data-block-id="4"><span><iframe></iframe></span></p><p data-block-id="5">This proxy for war-fighting ability isn’t the only one available, but I’d argue that it is the first one that should be reached for. It’s a proxy for being able to make things in general. Another possible one is <a href="https://en.wikipedia.org/wiki/List_of_countries_by_motor_vehicle_production">automobile manufacture</a>, as instruments of war tend to have complexities which are reasonably well approximated by the complexity of an automobile. Car manufacturers also make good contractors for complex weapons systems. You could argue that many countries don’t bother making native cars due to various technical/economic issues, or they have artificially propped up native car manufactures. The numbers also change more in times of conflict or economic problems; Russia’s car production was 2.25 million a year in times of peace and dropped to 3/4 of a million as the car plants were pressed into service making weapons. Russian steel manufacture was flat during the same time period. because it needs about the same amount of steel as before; about 70 megatons. As such, I prefer to simply look at the above list of how much steel is made.</p><p data-block-id="6"><img src="https://scottlocklin.wordpress.com/wp-content/uploads/2026/07/steel.jpg?w=720" alt=""></p><p data-block-id="7">I’d also look at <em>variance</em> in steel production as a useful second order proxy. If a country makes more steel in difficult times, this probably means they’re relatively stronger than the case where the opposite happens. In the former case it probably means they have all the resources needed to make things. In the latter case it probably means that they don’t, so if the oil and ore boats stop coming, they can’t make as much steel. So this denotes weakness. South Korea and Japan make less, and Turkey and Iran make more in hard times. But this is the cherry on top of our proxy model; you can leave it out, or just think about it a little bit (hurp derp, Japan not known for its iron ore), putting a little sharpie point on the weak sisters.</p><p data-block-id="8">Asking our foreign policy elite to notice things like steel production is, of course, futile. These are yalevard edumucated ninnyhammers who have more exotic spreadsheets which are capable of giving any answer their bosses require. The same educational colossus that netted us a fine arts major who <a href="https://www.reddit.com/r/SipsTea/comments/1tn7xeu/lupita_nyongo_was_unaware_of_what_the_odyssey/">never read Homer</a>. We’re not dealing with the best and brightest any more. None the less, these halfwits should overcome their hubris to avoid nemesis. All you need to do is look at the wikipedia page and avoid engaging in combats with countries in the “makes about half the steel that you’re capable of” column.</p><p data-block-id="9">There’s all kinds of stuff like this out there, most of it much more complicated than tons of steel. “Geopolitical thinkers” use all kinds of complex nonsense to rationalize bad ideas. <a href="https://www.analytickecentrum.cz/upload/soubor/original/measure-power.pdf">Ray Cline’s Equation</a> attempts to be quantitaive by adding together a bunch of qualitative factors. Easily gamed to produce the desired result. <a href="https://en.wikipedia.org/wiki/Hans_Morgenthau">Hans Morgenthau</a> same thing. J David Singer has a composite index of national capability which at least includes things like <a href="https://grokipedia.com/page/Composite_Index_of_National_Capability">steel production</a>, confounded with a bunch of stuff that either doesn’t mean anything or which is confounded by&nbsp; purchasing power parity. There are other things; people try to estimate <a href="https://warontherocks.com/you-go-to-war-with-the-industrial-base-you-have-not-the-industrial-base-you-want/">industrial base throughput</a>, critical materials supply chain, they develop <a href="https://en.wikipedia.org/wiki/Lanchester's_laws">differential equations</a>, various ideas about STEM graduates, gold holdings, strategic stockpiles, airlift/sealift capacity, civil-military fusion, strategic communication -all ideas which seem tailor made to keep dipshits in thinktanks fully employed writing reports that don’t mean anything. It’s tons of steel, the end. Just like it was when Paul Kennedy wrote <em><strong><span>The Rise and Fall of the Great Powers</span></strong></em><span>&nbsp;which more or less predicted the relative weakness of the US today.</span></p><div><img src="https://scottlocklin.wordpress.com/wp-content/uploads/2026/07/iranian-steel.jpg" alt=""><p data-block-id="10">supposedly Iranian steel mill</p></div><p data-block-id="11">Bringing it back around to military misadventures: of course Iran is giving the hyperpower a hard time. It makes 32 megatons of steel every year (trending upward in a wartime economy). The US makes 82. When you consider the home field advantage and the fact that they’re fighting for their lives rather than for …. nebulous reasons relating to domestic political donors, this poor performance was eminently predictable.</p></article>`

	enrichedHTML, err := feedsEnrichHTMLOutput(input)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(enrichedHTML)

	// mustEnv := func(key string) string {
	// 	val, ok := os.LookupEnv(key)
	// 	if !ok {
	// 		log.Fatalf("missing .env: %s", key)
	// 	}
	// 	return val
	// }

	// appDb := mustEnv("APP_DB")

	// dbHandle, err := sql.Open("sqlite3", appDb)
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }
	// dbHandle.SetMaxOpenConns(1)
	// dbHandle.SetMaxIdleConns(1)

	// _, _ = dbHandle.Exec(`PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000;`)

	// if err := dbHandle.Ping(); err != nil {
	// 	fmt.Println(err)
	// 	return
	// }

	// queries := db.New(dbHandle)

	// feeds, err := queries.SelectAllFeeds(context.Background())
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }

	// filteredA := []db.Feed{}

	// for _, v := range feeds {
	// 	if v.Url == "https://someurl.net" {
	// 		filteredA = append(filteredA, v)
	// 	}
	// }

	// fmt.Println(len(filteredA))

	// filteredB := lib.Filter(feeds, func(item db.Feed) bool {
	// 	return item.Url == "https://someurl.net"
	// })

	// fmt.Println(len(filteredB))

	// for i := range feeds {
	// 	feeds[i].Title = feeds[i].Title + "Asdfasdf"
	// }

	// godump.Dump(feeds[0].Title)

	// mapB := slices.Clone(feeds)
	// usefulTransforms := lib.Map(mapB, func(f db.Feed) string {
	// 	return strings.ToTitle(f.Title)
	// })

	// godump.Dump(usefulTransforms)

}
