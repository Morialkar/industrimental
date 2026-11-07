// Command balance plays an era without a screen, once per ending, and prints
// how long each one takes. With -curves it also writes the progression curves
// as one CSV file per ending.
package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/morialkar/industrimental/internal/balance"
	"github.com/morialkar/industrimental/internal/content"
)

func main() {
	eraID := flag.String("era", "e01", "era to simulate")
	curves := flag.String("curves", "", "directory for the CSV progression curves")
	verbose := flag.Bool("v", false, "print the milestones of each game")
	seed := flag.Uint64("seed", 1, "random seed")
	flag.Parse()

	b, err := content.LoadEmbedded()
	if err != nil {
		log.Fatal(err)
	}
	era, ok := b.Era(*eraID)
	if !ok {
		log.Fatalf("unknown era %q", *eraID)
	}
	failed := false
	for _, strat := range balance.Strategies() {
		r := balance.Play(era, strat, balance.Regular(), *seed)
		if r.Reached {
			fmt.Printf("%-11s portail à %5.1f h (Clé à %5.1f h), %d sessions\n", r.Axis, r.Hours, r.KeyHours, r.Sessions)
		} else {
			failed = true
			fmt.Printf("%-11s NON ATTEINT : %s\n", r.Axis, r.Left)
		}
		if *verbose {
			fmt.Println(r.Summary)
			for _, m := range r.Milestones {
				fmt.Printf("  %6.1f h  %s\n", m.Hours, m.What)
			}
		}
		if *curves != "" {
			if err := writeCurves(*curves, era, r); err != nil {
				log.Fatal(err)
			}
		}
	}
	if failed {
		os.Exit(1)
	}
}

func writeCurves(dir string, era *content.Era, r balance.Result) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(dir, string(r.Axis)+".csv"))
	if err != nil {
		return err
	}
	w := csv.NewWriter(f)
	header := []string{"hours", "stations", "nodes", "quests"}
	for _, res := range era.Resources {
		header = append(header, res.ID)
	}
	if err := w.Write(header); err != nil {
		_ = f.Close()
		return err
	}
	for _, s := range r.Samples {
		row := []string{num(s.Hours), strconv.Itoa(s.Stations), strconv.Itoa(s.Nodes), strconv.Itoa(s.Quests)}
		for _, res := range era.Resources {
			row = append(row, num(s.Stock[res.ID]))
		}
		if err := w.Write(row); err != nil {
			_ = f.Close()
			return err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func num(v float64) string { return strconv.FormatFloat(v, 'f', 2, 64) }
